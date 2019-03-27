// Copyright (c) 2016-2019, Andreas T Jonsson
// All rights reserved.

package build

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func Build() int {
	buildStart := time.Now()

	setupFlags()
	inputFiles := flag.Args()
	if len(inputFiles) < 1 {
		fmt.Fprintln(os.Stderr, "no input")
		return -1
	}

	os.Setenv("GOOS", "js")
	os.Setenv("GOARCH", "wasm")
	os.Setenv("GOROOT", goRoot)

	if workPath == "" {
		workPath = os.TempDir()
		defer os.RemoveAll(workPath)
	}

	workPath, _ = filepath.Abs(workPath)
	os.MkdirAll(workPath, 0755)

	tempWASMOutput := filepath.Join(workPath, "out.wasm")
	args := []string{
		"build",
		"-o", tempWASMOutput,
	}

	if len(buildTags) > 0 && !strings.HasPrefix(buildTags, " ") {
		buildTags = " " + buildTags
	}
	args = append(args, "-tags", "\"goc"+buildTags+"\"")

	inputFile, _ := filepath.Abs(inputFiles[0])
	args = append(args, inputFile)

	logln("Building Go code...")
	goBin := filepath.Join(goRoot, "bin", "go")
	if err := runProgram(goBin, filepath.Dir(inputFile), args...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return -1
	}

	logln("Generating C code...")
	tempCOutput := "out.c"
	wasm2cBin := filepath.Join(wabtPath, "wasm2c")
	if err := runProgram(wasm2cBin, workPath, tempWASMOutput, "-o", tempCOutput); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return -1
	}

	// Give C compiler absolute path.
	tempCOutput = filepath.Join(workPath, tempCOutput)

	rtCommonPath := filepath.Join(gocRoot, "wabt", "wasm2c")
	rtImplPath := filepath.Join(rtCommonPath, "wasm-rt-impl.c")

	cFiles := []string{
		tempCOutput,
		rtImplPath,
		filepath.Join(runtimePath, "rt.c"),
	}

	switch buildmode {
	case "exe":
		var cArgs, cTailArgs []string
		baseName := strings.TrimSuffix(strings.ToLower(filepath.Base(cCompiler)), ".exe")

		if baseName == "cl" {
			cArgs = []string{"/nologo", "/TP", "/DGOC_ENTRY=" + entryName, "/Fe" + outputName, "/I" + rtCommonPath, "/I", workPath}
		} else {
			cArgs = []string{"-std=c99", "-DGOC_ENTRY=" + entryName, "-o", outputName, "-I", rtCommonPath, "-I", workPath}
			if strings.Contains(baseName, "clang") {
				cTailArgs = []string{}
			} else {
				// Assume this is GCC.
				cTailArgs = []string{"-lm"}
			}
		}

		if cFlags != "" {
			for _, a := range strings.Split(cFlags, " ") {
				cArgs = append(cArgs, a)
			}
		}

		logln("Compiling C code...")
		finalArgs := append(append(cArgs, cFiles...), cTailArgs...)
		if err := runProgram(cCompiler, "", finalArgs...); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return -1
		}
	case "c-source":
		if err := os.MkdirAll(outputName, 0755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return -1
		}

		if err := copyFiles(outputName, workPath, "*.c *.h"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return -1
		}

		if err := copyFiles(outputName, runtimePath, "*.c *.h"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return -1
		}

		if err := copyFiles(outputName, rtCommonPath, "*.c *.h"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return -1
		}
	default:
		fmt.Fprintln(os.Stderr, "invalid buildmode:", buildmode)
		return -1
	}

	logln("Build time:", time.Since(buildStart).Round(time.Second))
	return 0
}

func runProgram(prog, cwd string, args ...string) error {
	print := []interface{}{cwd, prog}
	for _, s := range args {
		print = append(print, s)
	}
	logvln(print...)

	cmd := exec.Command(prog, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	cmd.Dir = cwd
	if err := cmd.Start(); err != nil {
		return err
	}

	slurp, err := ioutil.ReadAll(stderr)
	if err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return errors.New(string(slurp))
	}
	return nil
}

func copyFiles(destPath, path, globs string) error {
	for _, glob := range strings.Split(globs, " ") {
		glob := path + "/" + glob
		logvln("copyFiles:", glob, destPath)

		files, err := filepath.Glob(glob)
		if err != nil {
			return err
		}

		for _, file := range files {
			dest := filepath.Join(destPath, filepath.Base(file))
			logvln(file, "->", dest)

			srcFile, err := os.Open(file)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			destFile, err := os.Create(dest)
			if err != nil {
				return err
			}
			defer destFile.Close()

			if _, err = io.Copy(destFile, srcFile); err != nil {
				return err
			}
		}
	}
	return nil
}

func logln(a ...interface{}) {
	if !silent {
		fmt.Println(a...)
	}
}

func logvln(a ...interface{}) {
	if !silent && verbose {
		fmt.Println(a...)
	}
}

func About() string {
	return "build GopherC application"
}

var (
	cCompiler  = os.Getenv("CC")
	gocRoot    = os.Getenv("GOCROOT")
	outputName = "out"
	entryName  = "main"
	buildmode  = "exe"

	silent,
	verbose bool

	wabtPath,
	runtimePath,
	goRoot,
	workPath,
	cFlags,
	buildTags string
)

func setupFlags() {
	var exePath string
	if path, err := os.Executable(); err == nil {
		if final, err := filepath.EvalSymlinks(path); err == nil {
			path, _ = filepath.Abs(final)
		}
		exePath = filepath.Dir(path)
	}

	if runtime.GOOS == "windows" {
		outputName += ".exe"
	}

	if cCompiler == "" {
		cCompiler = "gcc"
	}

	flag.StringVar(&cCompiler, "cc", cCompiler, "set default C compiler (CC)")
	flag.StringVar(&buildTags, "tags", "", "a space-separated list of build tags")
	flag.StringVar(&wabtPath, "wabt", wabtPath, "wabt tools path")
	flag.StringVar(&goRoot, "goroot", goRoot, "Go compiler path")
	flag.StringVar(&gocRoot, "gocroot", gocRoot, "GopherC compiler path (GOCROOT)")
	flag.StringVar(&outputName, "o", outputName, "final output name")
	flag.StringVar(&entryName, "entry", entryName, "name of C entry point")
	flag.StringVar(&workPath, "work", workPath, "specify temporary work path")
	flag.StringVar(&cFlags, "cflags", cFlags, "extra parameters for the C compiler")
	flag.StringVar(&buildmode, "buildmode", buildmode, "set compiler buildmode, 'exe' or 'c-source'")
	flag.BoolVar(&silent, "s", silent, "silent mode")
	flag.BoolVar(&verbose, "v", verbose, "verbose")
	flag.Parse()

	if gocRoot == "" {
		gocRoot = filepath.Clean(filepath.Join(exePath, "..", ".."))
	}

	if goRoot == "" {
		goRoot = filepath.Join(gocRoot, "go")
	}

	if wabtPath == "" {
		wabtPath = filepath.Join(gocRoot, "wabt", "bin")
	}

	runtimePath = filepath.Join(gocRoot, "runtime")
}

func PrintDefaults() {
	setupFlags()
	fmt.Println("goc build [flags] [input]")
	flag.PrintDefaults()
}
