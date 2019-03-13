/*
Copyright (C) 2016-2019 Andreas T Jonsson

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package build

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Build() int {
	setupFlags()
	inputFiles := flag.Args()
	if len(inputFiles) < 1 {
		fmt.Fprintln(os.Stderr, "no input")
		return -1
	}

	os.Setenv("GOOS", "js")
	os.Setenv("GOARCH", "wasm")
	os.Setenv("GOROOT", goRoot)

	tempPath, _ := os.Getwd() // Temp path
	tempPath = filepath.Join(tempPath, "output")
	os.MkdirAll(tempPath, 0755)

	tempWASMOutput := filepath.Join(tempPath, "out.wasm")
	args := []string{
		"build",
		"-o", tempWASMOutput,
	}

	if len(buildTags) > 0 && !strings.HasPrefix(buildTags, " ") {
		buildTags = " " + buildTags
	}
	args = append(args, "-tags", "\"goc"+buildTags+"\"")

	args = append(args, inputFiles[0])

	logln("Building Go code...")
	goBin := filepath.Join(goRoot, "bin", "go")
	if err := runProgram(goBin, filepath.Dir(inputFiles[0]), args...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return -1
	}

	logln("Processing WASM...")
	tempCOutput := filepath.Join(tempPath, "out.c")
	wasm2cBin := filepath.Join(wabtPath, "wasm2c")
	if err := runProgram(wasm2cBin, tempPath, tempWASMOutput, "-o", tempCOutput); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return -1
	}

	logln("Compiling C code...")

	var err error
	if baseName := strings.ToLower(filepath.Base(cCompiler)); baseName == "cl" || baseName == "cl.exe" {
		err = runProgram(cCompiler, "", "/nologo", "/Fe"+outputName, "/I"+bindingsPath,
			tempCOutput,
			filepath.Join(bindingsPath, "wasm-rt-impl.c"),
			filepath.Join(runtimePath, "rt.c"),
		)
	} else {
		err = runProgram(cCompiler, "", "-std=c99", "-g", "-o", outputName, "-I", bindingsPath, "-I", tempPath,
			tempCOutput,
			filepath.Join(bindingsPath, "wasm-rt-impl.c"),
			filepath.Join(runtimePath, "rt.c"),
		)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return -1
	}
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

func logln(a ...interface{}) {
	if !verbose && !silent {
		fmt.Println(a...)
	}
}

func logvln(a ...interface{}) {
	if verbose {
		fmt.Println(a...)
	}
}

func About() string {
	return "build GopherC application"
}

var (
	cCompiler   = os.Getenv("CC")
	goRoot      = filepath.Clean(os.Getenv("GOROOT"))
	outputName  = "out.exe"
	runtimeName = "libc"

	silent,
	verbose bool

	wabtPath,
	bindingsPath,
	runtimePath,
	buildTags string
)

func setupFlags() {
	var exePath string
	if path, err := os.Executable(); err == nil {
		if final, err := filepath.EvalSymlinks(path); err == nil {
			path = final
		}
		exePath = filepath.Dir(path)
	}

	if cCompiler == "" {
		cCompiler = "gcc"
	}

	goRoot = filepath.Join(exePath, "go-bin")
	wabtPath = filepath.Join(exePath, "wabt-bin")
	bindingsPath = filepath.Join(exePath, "wabt", "wasm2c")

	flag.StringVar(&cCompiler, "cc", cCompiler, "set default C compiler (CC)")
	flag.StringVar(&buildTags, "tags", "", "a space-separated list of build tags")
	flag.StringVar(&bindingsPath, "bindings", bindingsPath, "path to stubs and bindings code")
	flag.StringVar(&wabtPath, "wabt", wabtPath, "wabt tools path")
	flag.StringVar(&goRoot, "goroot", goRoot, "Go compiler path (GOROOT)")
	flag.StringVar(&outputName, "o", outputName, "final output name")
	flag.StringVar(&runtimeName, "runtime", runtimeName, "runtime implementation")
	flag.BoolVar(&silent, "s", silent, "silent mode")
	flag.BoolVar(&verbose, "v", verbose, "verbose")
	flag.Parse()

	runtimePath = filepath.Join(bindingsPath, "..", "..", "runtime", runtimeName)
}

func PrintDefaults() {
	setupFlags()
	fmt.Println("goc build [flags] [input]")
	flag.PrintDefaults()
}
