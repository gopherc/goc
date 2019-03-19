// Copyright (c) 2016-2019, Andreas T Jonsson
// All rights reserved.

package main

import (
	"fmt"
)

func fib(n int) int {
	if n <= 1 {
		return n
	}
	return fib(n-2) + fib(n-1)
}

func main() {
	const n = 40
	fmt.Println("Fib:", n, fib(n))
}
