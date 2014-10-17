package main

import (
	"crypto/tls"
	"fmt"
	"github.com/cloudfoundry/noaa"
	"os"
)

var DopplerAddress = "wss://doppler.10.244.0.34.xip.io:443"
var authToken = os.Getenv("CF_ACCESS_TOKEN")

func main() {
	connection := noaa.NewNoaa(DopplerAddress, &tls.Config{InsecureSkipVerify: true}, nil)
	connection.SetDebugPrinter(ConsoleDebugPrinter{})

	fmt.Println("===== Streaming Firehose (will only succeed if you have admin credentials)")
	msgChan, err := connection.Firehose(authToken)

	if err != nil {
		fmt.Printf("===== Error streaming: %v\n", err)
	} else {
		for msg := range msgChan {
			fmt.Printf("%v \n", msg)
		}
	}

}

type ConsoleDebugPrinter struct{}

func (c ConsoleDebugPrinter) Print(title, dump string) {
	println(title)
	println(dump)
}
