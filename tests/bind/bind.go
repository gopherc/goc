// Copyright (c) 2016-2019, Andreas T Jonsson
// All rights reserved.

package main

import (
	"fmt"

	"github.com/gopherc/goc/tests/bind/bind"
)

func main() {
	// Should print: !33
	fmt.Println(bind.Putc(33))
}
