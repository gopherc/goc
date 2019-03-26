// Copyright (c) 2016-2019, Andreas T Jonsson
// All rights reserved.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const (
	benchmark   = true
	conformance = false
)

func main() {
	//os.Setenv("CC", "clang")

	knucleotide, err := ioutil.ReadFile("k-nucleotide-input.txt")
	if err != nil {
		panic(err)
	}

	if conformance {
		test("../fib/fib.go", nil)
		test("../fiber/fiber.go", nil)
		test("../hello/hello.go", nil)
		test("../resize/resize.go", nil)
	}

	if benchmark {
		test("../garbage/garbage.go", nil, "20000000")
		test("../mandelbrot/mandelbrot.go", nil, "16000")
		test("../reverse-complement/reverse.go", knucleotide)
		//test("../k-nucleotide/knucleotide.go", knucleotide)
		test("../n-body/nbody.go", nil, "50000000")
	}
}

func test(src string, input []byte, args ...string) {
	if err := runTest(src, input, args...); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func runTest(src string, input []byte, args ...string) error {
	var suffix = ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}

	wd := filepath.Dir(src)
	base := filepath.Base(src)
	output := filepath.Join(wd, "go_"+base+suffix)
	if err := runProgram("../../go/bin/go"+suffix, wd, nil, "build", "-o", output, src); err != nil {
		return err
	}
	defer os.Remove(output)

	t := time.Now()
	if err := runProgram(output, wd, input, args...); err != nil {
		return err
	}
	fmt.Println("[go]", base+":", time.Since(t).Round(time.Millisecond))

	f := func(flags string) error {
		wd := filepath.Dir(src)
		base := filepath.Base(src)
		output := filepath.Join(wd, "goc_"+base+suffix)

		t := time.Now()
		if err := runProgram("../../cmd/goc/goc"+suffix, wd, nil, "build", "-cflags="+flags, "-o", output, src); err != nil {
			return err
		}
		buildTime := time.Since(t).Round(time.Second)
		defer os.Remove(output)

		t = time.Now()
		if err := runProgram(output, wd, input, args...); err != nil {
			return err
		}
		fmt.Printf("[goc %s] %s: %v (%v)\n", flags, base, time.Since(t).Round(time.Millisecond), buildTime)

		return nil
	}

	if err := f("-O0"); err != nil {
		return err
	}

	if benchmark {
		if err := f("-O1"); err != nil {
			return err
		}

		if err := f("-O2"); err != nil {
			return err
		}

		if err := f("-O3"); err != nil {
			return err
		}
	}
	return nil
}

func runProgram(prog, wd string, input []byte, args ...string) error {
	cmd := exec.Command(prog, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if input != nil {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		go func() {
			io.Copy(stdin, bytes.NewReader(input))
			stdin.Close()
		}()
	}

	cmd.Dir = wd
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
