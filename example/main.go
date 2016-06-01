package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xilixsys/go-native-ads"
)

func main() {
	version := ads.AdsGetDllVersion()
	log.Println(version.Version, version.Revision, version.Build)

	fmt.Println()

	port := ads.AdsPortOpen()
	log.Println(port)
	address := ads.AddLocalConnection()

	variable := address.Symbols["MAIN.i"]
	val, err := variable.GetStringValue()
	variable.Write("15", 0)
	fmt.Println(variable.GetStringValue())
	variable.AdsSyncAddDeviceNotificationReq(0, 0, 0)

	fmt.Println("error", err)
	fmt.Println("value", val)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	go func() {
		<-c
		address.CloseEverything()
		// sig is a ^C, handle it
		os.Exit(1)
	}()

	for {

	}
}
