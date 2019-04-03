// Copyright (c) 2016-2019, Andreas T Jonsson
// All rights reserved.

package main

import (
	"fmt"
	"os"

	"github.com/gopherc/goc/cmd/goc/bind"
	"github.com/gopherc/goc/cmd/goc/build"
	"github.com/gopherc/goc/cmd/goc/version"
)

func main() {
	if len(os.Args) < 2 {
		printHeader()
		fmt.Fprintln(os.Stderr, "Tools:")
		printTools()
		os.Exit(-1)
	}

	i := 1
	sz := 1
	tool := os.Args[1]
	for _, a := range os.Args[2:] {
		if a != "-h" {
			os.Args[i] = a
			sz++
			i++
		}
	}
	os.Args = os.Args[:sz]

	switch tool {
	case "bind":
		os.Exit(bind.Bind())
	case "build":
		os.Exit(build.Build())
	case "version":
		fmt.Println("GopherC version:", version.Version)
	case "help", "-help", "-h":
		if len(os.Args) == 2 {
			printToolHelp(os.Args[1])
			return
		}
		printHeader()
		printTools()
	default:
		fmt.Fprintln(os.Stderr, "Invalid tool:", tool)
		printTools()
		os.Exit(-1)
	}
}

func printHeader() {
	fmt.Println("goc - GopherC compiler\nCopyright (C) 2016-2019 Andreas T Jonsson\n")
	fmt.Println("You can run 'goc help [tool]' for more information of a specific tool.\n")
}

func printToolHelp(tool string) {
	switch tool {
	case "bind":
		bind.PrintDefaults()
	case "build":
		build.PrintDefaults()
	default:
		fmt.Println(tool, "has no options")
	}
}

func printTools() {
	fmt.Println("\tbind\t" + bind.About())
	fmt.Println("\tbuild\t" + build.About())
	fmt.Println("\thelp\tlist tools and options")
	fmt.Println("\tversion\t" + version.About())
}
