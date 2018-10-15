package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.com/xilix-systems-llc/go-native-ads/v2"
)

func main() {
	version := ads.AdsGetDllVersion()
	log.Println(version.Version, version.Revision, version.Build)

	fmt.Println()

	address, _ := ads.AddLocalConnection()

	variable := address.Symbols["GVL.TakePicture"]
	// for _, child := range variable.Childs {
	// 	fmt.Println(child.Name, child.FullName)
	// }
	// for key, variable := range address.Symbols {
	// 	fmt.Println(key, variable.FullName)

	// }

	variable.AddNotification(4, uint32(time.Second)/4, uint32(time.Second)/4, sendJson)

	// jsonObj := gabs.New()
	// variable.GetJson(jsonObj, "")
	// fmt.Println(jsonObj.StringIndent("", "  "))

	// val, err := variable.GetStringValue()
	// variable.Write("15", 0)
	// fmt.Println(variable.GetStringValue())

	// fmt.Println("error", err)
	// fmt.Println("value", val)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	<-c
	ads.CloseAllConnections()
	fmt.Println("closing")
	// sig is a ^C, handle it
	os.Exit(1)
}

func sendJson(symbol ads.ADSSymbol) {
	// fmt.Println("Callback", symbol.Name, symbol.Value)
	jsonReturn, err := symbol.GetJSON(true)
	if err == nil {
		fmt.Println(string(jsonReturn))
	} else {
		fmt.Println(err)
	}
}
