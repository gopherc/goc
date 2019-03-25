// Copyright (c) 2016-2019, Andreas T Jonsson
// All rights reserved.

package main

import (
	"flag"
	"image"
	"image/png"
	"os"

	"github.com/nfnt/resize"
)

var (
	width  uint = 1920
	height uint = 1080
)

func main() {
	flag.UintVar(&width, "x", width, "output image width")
	flag.UintVar(&height, "y", height, "output image height")
	flag.Parse()

	src, err := png.Decode(os.Stdin)
	if err != nil {
		panic(err)
	}

	res := resize.Resize(width, height, src, resize.Lanczos3).(*image.RGBA)
	if err := png.Encode(os.Stdout, res); err != nil {
		panic(err)
	}
}
