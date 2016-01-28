package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Printf("I am working at %v o'clock\n", time.Now())
		time.Sleep(1 * time.Second)
	}
}
