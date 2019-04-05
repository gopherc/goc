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
	Import, Include               string
	Conversion                    string
}

type FuncBinding struct {
	Comment     string
	Call        string
	Args        []string
	Ret         string
	Block, Safe bool
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
	var goImports, cIncludes map[string]struct{}

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

			if info.Name() == "goc.type" {
				fp, err := os.Open(path)
				if err != nil {
					return err
				}
				defer fp.Close()

				var types map[string]TypeSpec
				dec := json.NewDecoder(fp)
				if err := dec.Decode(&types); err != nil {
					return err
				}

				for name, ty := range types {
					if ty.InternalGoType == "" {
						ty.InternalGoType = ty.GoType
					}
					if ty.Import != "" {
						goImports[ty.Import] = struct{}{}
					}
					if ty.Include != "" {
						cIncludes[ty.Include] = struct{}{}
					}
					allTypes[filepath.Join(pkgPath, name)] = ty
				}
			}
		}

		return nil
	}); err != nil {
		return err
	}

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

	if len(cIncludes) > 0 {
		for inc := range cIncludes {
			fmt.Fprintf(fpc, "#include <%s>\n", inc)
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

			if info.Name() == "goc.bind" {
				fp, err := os.Open(path)
				if err != nil {
					return err
				}
				defer fp.Close()

				var bindings map[string]FuncBinding
				dec := json.NewDecoder(fp)
				if err := dec.Decode(&bindings); err != nil {
					return err
				}

				if err := generateGoTrampoline(fpc, pkgPath, filepath.Dir(path), bindings, goImports); err != nil {
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

func generateGoTrampoline(fpc io.Writer, pkgPath, path string, bindings map[string]FuncBinding, imports map[string]struct{}) error {
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

	fmt.Fprint(fps, "// +build goc\n\n")
	fmt.Fprint(fps, "// Generated by the GopherC bind tool.\n")
	fmt.Fprintf(fps, "// %v\n\n", time.Now())
	fmt.Fprintf(fps, "#include \"textflag.h\"\n\n")

	fmt.Fprint(fpg, "// Generated by the GopherC bind tool.\n")
	fmt.Fprintf(fpg, "// %v\n\n", time.Now())
	fmt.Fprint(fpg, "// +build goc\n\n")
	fmt.Fprintf(fpg, "package %s\n\n", pkg)

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
		if len(bind.Args)%2 != 0 {
			fmt.Fprintf(os.Stderr, "%s has invalid arguments\n", fullFuncName)
		}

		fmt.Fprintf(fps, "TEXT ·goc%s(SB), NOSPLIT, $0\n\tCallImport\n\tRET\n\n", funcName)

		fmt.Fprintf(fpg, "func goc%s(", funcName)

		writeArgs := func(prefix string, internal bool) {
			for i := 0; i < len(bind.Args); i += 2 {
				name := bind.Args[i]
				ty := bind.Args[i+1]

				if i > 0 {
					fmt.Fprint(fpg, ", ")
				}

				if prefix != "" {
					fmt.Fprint(fpg, prefix+name)
				} else {
					fmt.Fprintf(fpg, "%s ", name)

					fullName := filepath.Join(pkgPath, ty)
					typeSpec, ok := allTypes[fullName]
					if !ok {
						fmt.Fprintf(os.Stderr, "%s has invalid argument type: %s\n", fullFuncName, fullName)
					}

					if internal {
						fmt.Fprint(fpg, typeSpec.InternalGoType)
					} else {
						fmt.Fprint(fpg, typeSpec.GoType)
					}
				}
			}
		}

		writeArgs("", true)
		fmt.Fprint(fpg, ")")

		if bind.Ret != "" {
			fullName := filepath.Join(pkgPath, bind.Ret)
			typeSpec, ok := allTypes[fullName]
			if !ok {
				fmt.Fprintf(os.Stderr, "%s has invalid return argument type: %s\n", fullFuncName, fullName)
			}
			fmt.Fprintf(fpg, " %s\n\n", typeSpec.InternalGoType)
		} else {
			fmt.Fprint(fpg, "\n\n")
		}

		if bind.Comment != "" {
			fmt.Fprintf(fpg, "// %s\n", bind.Comment)
		}

		fmt.Fprintf(fpg, "func %s(", funcName)
		writeArgs("", false)

		if bind.Ret != "" {
			fullName := filepath.Join(pkgPath, bind.Ret)
			typeSpec := allTypes[fullName]
			fmt.Fprintf(fpg, ") %s {\n", typeSpec.GoType)
		} else {
			fmt.Fprint(fpg, ") {\n")
		}

		for i := 0; i < len(bind.Args); i += 2 {
			name := bind.Args[i]
			ty := bind.Args[i+1]

			fullName := filepath.Join(pkgPath, ty)
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

		writeArgs("_", false)
		fmt.Fprint(fpg, ")\n")

		if bind.Ret != "" {
			fullName := filepath.Join(pkgPath, bind.Ret)
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

		if err := generateCTrampoline(fpc, pkgPath, funcName, bind); err != nil {
			return err
		}
	}

	return nil
}

func generateCTrampoline(fp io.Writer, pkgPath, funcName string, bind FuncBinding) error {
	var retCType string
	if bind.Ret != "" {
		fullName := filepath.Join(pkgPath, bind.Ret)
		retCType = allTypes[fullName].CType

		fmt.Fprintf(fp, "extern %s %s(", retCType, bind.Call)
	} else {
		fmt.Fprintf(fp, "extern void %s(", bind.Call)
	}

	for i := 0; i < len(bind.Args); i += 2 {
		fullName := filepath.Join(pkgPath, bind.Args[i+1])
		typeSpec := allTypes[fullName]
		if i > 0 {
			fmt.Fprint(fp, ", ")
		}
		fmt.Fprintf(fp, "%s", typeSpec.CType)
	}
	fmt.Fprint(fp, ");\n")

	mangledName := mangleCName(filepath.Join(moduleName, pkgPath), "goc"+funcName)
	fmt.Fprintf(fp, "static void _%s(uint32_t sp) {\n", mangledName)

	fmt.Fprint(fp, "\tsp += 8;\n")
	for i := 0; i < len(bind.Args); i += 2 {
		name := bind.Args[i]
		ty := bind.Args[i+1]

		fullName := filepath.Join(pkgPath, ty)
		typeSpec := allTypes[fullName]

		fmt.Fprintf(fp, "\t%s _%s = *(%s*)&Z_mem->data[sp];\n", typeSpec.CType, name, typeSpec.CType)
		fmt.Fprintf(fp, "\tsp += sizeof(%s) + ((8 - (sizeof(%s) %% 8)) %% 8);\n", typeSpec.CType, typeSpec.CType)
	}

	if retCType != "" {
		fmt.Fprintf(fp, "\t%s _r = %s(", retCType, bind.Call)
	} else {
		fmt.Fprintf(fp, "\t%s(", bind.Call)
	}

	for i := 0; i < len(bind.Args); i += 2 {
		if i > 0 {
			fmt.Fprint(fp, ", ")
		}
		fmt.Fprintf(fp, "_%s", bind.Args[i])
	}
	fmt.Fprint(fp, ");\n")

	if retCType != "" {
		fmt.Fprintf(fp, "\tmemcpy(&Z_mem->data[sp], &_r, sizeof(%s));\n", retCType)
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
	cBindFile  string

	Silent,
	Verbose bool
)

func setupFlags() {
	flag.StringVar(&moduleName, "m", moduleName, "module name")
	flag.StringVar(&cBindFile, "o", cBindFile, "resulting C binding file")
	flag.BoolVar(&Silent, "s", Silent, "silent mode")
	flag.BoolVar(&Verbose, "v", Verbose, "verbose")
	flag.Parse()
}

func PrintDefaults() {
	setupFlags()
	fmt.Println("goc bind [path]")
	flag.PrintDefaults()
}
