// Copyright (c) 2016-2019, Andreas T Jonsson
// All rights reserved.

package main

import (
	"fmt"
	"time"
)

func main() {
	go func() {
		for i := 0; i < 10; i++ {
			fmt.Println("Gorutine 2, loop:", i)
			time.Sleep(time.Second)
		}
		fmt.Println("Gorutine 2 done!")
	}()

	for i := 0; i < 100; i++ {
		fmt.Println("Gorutine 1, loop:", i)
		time.Sleep(100 * time.Millisecond)
	}
}
