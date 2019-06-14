// Copyright (c) 2016-2019, Andreas T Jonsson
// All rights reserved.

package bind

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

type TypeSpec struct {
	GoType, InternalGoType, CType string
	Conversion                    string
	CRef, CPush                   string
	Align	                      int
	SkipImport                    bool
	GoImports, CDeclarations      []string
}

type FuncBinding struct {
	Comment          string
	Call             string
	Args             []string
	Ret              string
	CPrefix, CSuffix string

	Extern,
	Member,
	BindOnly bool
}

var allTypes = map[string]TypeSpec{}

func Bind() int {
	setupFlags()
	inputPaths := flag.Args()
	if len(inputPaths) < 1 {
		fmt.Fprintln(os.Stderr, "no source")
		return -1
	}

	output := cBindFile
	if output == "" {
		output = filepath.Join(inputPaths[0], "bind_goc.c")
	}

	if err := Generate(inputPaths[0], output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return -1
	}
	return 0
}

func Generate(projectPath, outputFile string) error {
	var cDeclarations []string

	getDir := func(base, path string) (string, error) {
		pd := filepath.Dir(path)
		rel, err := filepath.Rel(base, pd)
		if err != nil {
			return "", err
		}
		return filepath.Clean(rel), nil
	}

	// Start looking for types.
	if err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			pkgPath, err := getDir(projectPath, path)
			if err != nil {
				return err
			}
			pkgPath = filepath.Join(moduleName, pkgPath)

			if info.Name() == "goc.type" {
				fp, err := os.Open(path)
				if err != nil {
					return err
				}
				defer fp.Close()

				var types map[string]TypeSpec
				dec := json.NewDecoder(fp)

				logvln("Decode:", path)
				if err := dec.Decode(&types); err != nil {
					return err
				}

				for name, ty := range types {
					if ty.InternalGoType == "" {
						ty.InternalGoType = ty.GoType
					}
					for _, dec := range ty.CDeclarations {
						cDeclarations = append(cDeclarations, dec)
					}

					cleanName := filepath.Join(pkgPath, name)
					if strings.HasPrefix(name, ".") {
						cleanName = strings.TrimPrefix(name, ".")
					} else if !ty.SkipImport {
						imp := strings.ReplaceAll(pkgPath, "\\", "/")
						ty.GoImports = append(ty.GoImports, imp)
					}

					cleanName = strings.ReplaceAll(cleanName, "\\", "/")
					allTypes[cleanName] = ty
				}
			}
		}

		return nil
	}); err != nil {
		return err
	}

	logvln("All types: [")
	for name := range allTypes {
		logvln(name)
	}
	logvln("]")

	fpc, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer fpc.Close()

	fmt.Fprint(fpc, "// Generated by the GopherC bind tool.\n")
	fmt.Fprintf(fpc, "// %v\n\n", time.Now())
	fmt.Fprint(fpc, "#ifdef __cplusplus\nextern \"C\" {\n#endif\n\n")
	fmt.Fprint(fpc, "#include <string.h>\n")
	fmt.Fprint(fpc, "#include <wasm-rt.h>\n\n")

	if len(cDeclarations) > 0 {
		for _, dec := range cDeclarations {
			fmt.Fprintln(fpc, dec)
		}
		fmt.Fprint(fpc, "\n")
	}

	fmt.Fprint(fpc, "extern uint32_t (*Z_getspZ_iv)();\n")
	fmt.Fprint(fpc, "extern wasm_rt_memory_t *Z_mem;\n\n")

	// Search for bindings.
	if err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			pkgPath, err := getDir(projectPath, path)
			if err != nil {
				return err
			}
			pkgPath = filepath.Join(moduleName, pkgPath)

			if info.Name() == "goc.bind" {
				fp, err := os.Open(path)
				if err != nil {
					return err
				}
				defer fp.Close()

				var bindings map[string]FuncBinding
				dec := json.NewDecoder(fp)

				logvln("Decode:", path)
				if err := dec.Decode(&bindings); err != nil {
					return err
				}

				if err := generateGoTrampoline(fpc, pkgPath, filepath.Dir(path), bindings); err != nil {
					return nil
				}
			}
		}

		return nil
	}); err != nil {
		return err
	}

	fmt.Fprint(fpc, "#ifdef __cplusplus\n}\n#endif\n")
	return nil
}

func createTypePath(pkgPath, ty string) string {
	local := strings.ReplaceAll(filepath.Join(pkgPath, ty), "\\", "/")
	if _, ok := allTypes[local]; ok {
		return local
	}
	return ty
}

func createTypeName(pkg, name string) string {
	pkgName := filepath.Base(pkg)
	tyName := filepath.Base(name)
	return strings.TrimPrefix(tyName, pkgName+".")
}

func generateGoTrampoline(fpc io.Writer, pkgPath, path string, bindings map[string]FuncBinding) error {
	logvln("Generate:", pkgPath)

	fpg, err := os.Create(filepath.Join(path, "bind_goc.go"))
	if err != nil {
		return err
	}
	defer fpg.Close()

	fps, err := os.Create(filepath.Join(path, "bind_goc.s"))
	if err != nil {
		return err
	}
	defer fps.Close()

	pkg := filepath.Base(pkgPath)

	if buildTags != "" {
		fmt.Fprintf(fps, "// +build %s\n\n", buildTags)
	}

	fmt.Fprint(fps, "// Generated by the GopherC bind tool.\n")
	fmt.Fprintf(fps, "// %v\n\n", time.Now())
	fmt.Fprintf(fps, "#include \"textflag.h\"\n\n")

	fmt.Fprint(fpg, "// Generated by the GopherC bind tool.\n")
	fmt.Fprintf(fpg, "// %v\n\n", time.Now())
	
	if buildTags != "" {
		fmt.Fprintf(fpg, "// +build %s\n\n", buildTags)
	}

	fmt.Fprintf(fpg, "package %s\n\n", pkg)

	imports := map[string]struct{}{}
	addImports := func(lst []string) {
		cleanPkg := strings.ReplaceAll(pkgPath, "\\", "/")
		for _, imp := range lst {
			if cleanPkg != imp {
				imports[imp] = struct{}{}
			}
		}
	}

	for funcName, bind := range bindings {
		fullFuncName := filepath.Join(pkgPath, funcName)
		if len(bind.Args)%2 != 0 {
			fmt.Fprintf(os.Stderr, "%s has invalid arguments\n", fullFuncName)
		}

		for i := 0; i < len(bind.Args); i += 2 {
			ty := bind.Args[i+1]
			fullName := createTypePath(pkgPath, ty)
			typeSpec, ok := allTypes[fullName]
			if !ok {
				fmt.Fprintf(os.Stderr, "%s has invalid argument type: %s\n", fullFuncName, fullName)
			}
			addImports(typeSpec.GoImports)
		}

		if bind.Ret != "" {
			fullName := createTypePath(pkgPath, bind.Ret)
			typeSpec, ok := allTypes[fullName]
			if !ok {
				fmt.Fprintf(os.Stderr, "%s has invalid return argument type: %s\n", fullFuncName, fullName)
			}
			addImports(typeSpec.GoImports)
		}
	}

	if len(imports) > 0 {
		fmt.Fprint(fpg, "import (\n")
		for imp := range imports {
			fmt.Fprintf(fpg, "\t\"%s\"\n", imp)
		}
		fmt.Fprint(fpg, ")\n\n")
	}

	for funcName, bind := range bindings {
		logvln(funcName, "->", bind.Call)
		fullFuncName := filepath.Join(pkgPath, funcName)

		fmt.Fprintf(fps, "TEXT ·goc%s(SB), NOSPLIT, $0\n\tCallImport\n\tRET\n\n", funcName)

		fmt.Fprintf(fpg, "func goc%s(", funcName)

		writeArgs := func(prefix string, start int, internal bool) {
			for i := start; i < len(bind.Args); i += 2 {
				name := bind.Args[i]
				ty := bind.Args[i+1]

				if i > start {
					fmt.Fprint(fpg, ", ")
				}

				if prefix != "" {
					fmt.Fprint(fpg, prefix+name)
				} else {
					fmt.Fprintf(fpg, "%s ", name)

					fullName := createTypePath(pkgPath, ty)
					typeSpec := allTypes[fullName]
					tyName := createTypeName(pkgPath, typeSpec.GoType)

					if internal {
						tyName = createTypeName(pkgPath, typeSpec.InternalGoType)
					}
					fmt.Fprint(fpg, tyName)
				}
			}
		}

		writeArgs("", 0, true)
		fmt.Fprint(fpg, ")")

		if bind.Ret != "" {
			fullName := createTypePath(pkgPath, bind.Ret)
			typeSpec := allTypes[fullName]
			fmt.Fprintf(fpg, " %s\n\n", createTypeName(pkgPath, typeSpec.InternalGoType))
		} else {
			fmt.Fprint(fpg, "\n\n")
		}

		if !bind.BindOnly {
			if bind.Comment != "" {
				fmt.Fprintf(fpg, "// %s\n", bind.Comment)
			}

			if bind.Member {
				if len(bind.Args) < 2 {
					fmt.Fprintf(os.Stderr, "%s has no arguments\n", fullFuncName)
				}

				fullName := createTypePath(pkgPath, bind.Args[1])
				typeSpec := allTypes[fullName]
				fmt.Fprintf(fpg, "func (%s %s) %s(", bind.Args[0], createTypeName(pkgPath, typeSpec.GoType), funcName)

				writeArgs("", 2, false)
			} else {
				fmt.Fprintf(fpg, "func %s(", funcName)
				writeArgs("", 0, false)
			}

			if bind.Ret != "" {
				fullName := createTypePath(pkgPath, bind.Ret)
				typeSpec := allTypes[fullName]
				fmt.Fprintf(fpg, ") %s {\n", createTypeName(pkgPath, typeSpec.GoType))
			} else {
				fmt.Fprint(fpg, ") {\n")
			}

			for i := 0; i < len(bind.Args); i += 2 {
				name := bind.Args[i]
				ty := bind.Args[i+1]

				fullName := createTypePath(pkgPath, ty)
				typeSpec := allTypes[fullName]

				conv := strings.ReplaceAll(typeSpec.Conversion, "@", name)
				if conv == "" {
					conv = name
				}
				fmt.Fprintf(fpg, "\t_%s := %s\n", name, conv)
			}

			if bind.Ret != "" {
				fmt.Fprintf(fpg, "\t_r := goc%s(", funcName)
			} else {
				fmt.Fprintf(fpg, "\tgoc%s(", funcName)
			}

			writeArgs("_", 0, false)
			fmt.Fprint(fpg, ")\n")

			if bind.Ret != "" {
				fullName := createTypePath(pkgPath, bind.Ret)
				typeSpec, ok := allTypes[fullName]
				if !ok {
					fmt.Fprintf(os.Stderr, "%s has invalid return argument type: %s\n", fullFuncName, fullName)
				}

				if typeSpec.Conversion == "" {
					fmt.Fprintf(fpg, "\treturn _r\n")
				} else {
					fmt.Fprintf(fpg, "\treturn %s\n", strings.ReplaceAll(typeSpec.Conversion, "@", "_r"))
				}
			}

			fmt.Fprint(fpg, "}\n\n")
		}

		if err := generateCTrampoline(fpc, pkgPath, funcName, bind); err != nil {
			return err
		}
	}

	return nil
}

func generateCTrampoline(fp io.Writer, pkgPath, funcName string, bind FuncBinding) error {
	var (
		retCType string
		retSpec  TypeSpec
	)

	fmt.Fprintf(fp, "// %s.%s -> %s\n", pkgPath, funcName, bind.Call)

	if bind.Ret != "" {
		fullName := createTypePath(pkgPath, bind.Ret)
		retSpec = allTypes[fullName]
		retCType = retSpec.CType
	}

	if bind.Extern {
		if retCType != "" {
			fmt.Fprintf(fp, "extern %s %s(", retCType, bind.Call)
		} else {
			fmt.Fprintf(fp, "extern void %s(", bind.Call)
		}

		for i := 0; i < len(bind.Args); i += 2 {
			fullName := createTypePath(pkgPath, bind.Args[i+1])
			typeSpec := allTypes[fullName]
			if i > 0 {
				fmt.Fprint(fp, ", ")
			}
			fmt.Fprintf(fp, "%s", typeSpec.CType)
		}
		fmt.Fprint(fp, ");\n")
	}

	mangledName := mangleCName(pkgPath, "goc"+funcName)
	fmt.Fprintf(fp, "static void _%s(uint32_t sp) {\n", mangledName)

	if bind.CPrefix != "" {
		fmt.Fprintf(fp, "\t%s\n", bind.CPrefix)
	}

	fmt.Fprint(fp, "\tsp += 8;\n")
	for i := 0; i < len(bind.Args); i += 2 {
		name := bind.Args[i]
		ty := bind.Args[i+1]

		fullName := createTypePath(pkgPath, ty)
		typeSpec := allTypes[fullName]

		align := fmt.Sprintf("sizeof(%s)", typeSpec.CType)
		if typeSpec.Align > 0 {
			align = fmt.Sprintf("%d", typeSpec.Align)
		}

		fmt.Fprintf(fp, "\tsp = (sp + (%s - 1)) & -%s;\n", align, align)
		fmt.Fprintf(fp, "\t%s _%s = *(%s*)&Z_mem->data[sp];\n", typeSpec.CType, name, typeSpec.CType)
		fmt.Fprintf(fp, "\tsp += sizeof(%s);\n", typeSpec.CType)
	}

	if retCType != "" {
		fmt.Fprintf(fp, "\t%s _r = %s(", retCType, bind.Call)
	} else {
		fmt.Fprintf(fp, "\t%s(", bind.Call)
	}

	for i := 0; i < len(bind.Args); i += 2 {
		fullName := createTypePath(pkgPath, bind.Args[i+1])
		typeSpec := allTypes[fullName]

		if i > 0 {
			fmt.Fprint(fp, ", ")
		}
		if typeSpec.CRef != "" {
			fmt.Fprint(fp, strings.ReplaceAll(typeSpec.CRef, "@", "_"+bind.Args[i]))
		} else {
			fmt.Fprintf(fp, "_%s", bind.Args[i])
		}
	}
	fmt.Fprint(fp, ");\n")

	if retCType != "" {
		fmt.Fprint(fp, "\tsp = (sp + (8 - 1)) & -8;\n")
		if retSpec.CPush != "" {
			fmt.Fprintf(fp, "%s\n", strings.ReplaceAll(retSpec.CPush, "@", "_r"))
		} else {
			fmt.Fprintf(fp, "\tmemcpy(&Z_mem->data[sp], &_r, sizeof(%s));\n", retCType)
		}
	}

	if bind.CSuffix != "" {
		fmt.Fprintf(fp, "\t%s\n", bind.CSuffix)
	}

	fmt.Fprintf(fp, "}\nvoid (*%s)(uint32_t) = _%s;\n\n", mangledName, mangledName)
	return nil
}

func mangleCName(dir, name string) string {
	result := "Z_goZ_"
	for _, c := range fmt.Sprintf("%s.%s", strings.ReplaceAll(dir, "\\", "/"), name) {
		if ((unicode.IsLetter(c) || unicode.IsNumber(c)) && c != 'Z') || c == '_' {
			result += string(c)
		} else {
			result = fmt.Sprintf("%sZ%X", result, uint8(c))
		}
	}
	return result + "Z_vi"
}

func logln(a ...interface{}) {
	if !Silent {
		fmt.Println(a...)
	}
}

func logvln(a ...interface{}) {
	if !Silent && Verbose {
		fmt.Println(a...)
	}
}

func About() string {
	return "generate C bindings"
}

var (
	moduleName = "github.com/user/mod"
	buildTags = "goc"
	cBindFile  string

	Silent,
	Verbose bool
)

func setupFlags() {
	flag.StringVar(&moduleName, "m", moduleName, "module name")
	flag.StringVar(&cBindFile, "o", cBindFile, "resulting C binding file")
	flag.StringVar(&buildTags, "tags", buildTags, "build tags")
	flag.BoolVar(&Silent, "s", Silent, "silent mode")
	flag.BoolVar(&Verbose, "v", Verbose, "verbose")
	flag.Parse()
}

func PrintDefaults() {
	setupFlags()
	fmt.Println("goc bind [path]")
	flag.PrintDefaults()
}
