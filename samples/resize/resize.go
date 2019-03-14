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
