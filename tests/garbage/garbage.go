// Copyright (c) 2016-2019, Andreas T Jonsson
// All rights reserved.

package main

import (
	"math/rand"
	"os"
	"strconv"
)

func main() {
	data := map[int][]byte{}
	n, _ := strconv.Atoi(os.Args[1])
	for i := 0; i < n; i++ {
		sz := rand.Intn(100)
		d := make([]byte, sz)
		rand.Read(d)
		data[sz] = d
	}
	data = nil
}
