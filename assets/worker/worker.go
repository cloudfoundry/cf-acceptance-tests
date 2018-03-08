package main

import (
	"fmt"
	"time"
)

func main() {
	for i := 1; ; i++ {
		fmt.Println("Running Worker", i)
		time.Sleep(time.Second)
	}
}
