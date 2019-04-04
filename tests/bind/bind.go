// Copyright (c) 2016-2019, Andreas T Jonsson
// All rights reserved.

package main

import (
	"fmt"

	"github.com/gopherc/goc/tests/bind/bind"
)

func main() {
	fmt.Println(bind.Putc(33)) // Should print: !33
}
