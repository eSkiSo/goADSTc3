package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xilixsys/go-native-ads"
)

func main() {
	version := ads.AdsGetDllVersion()
	log.Println(version.Version, version.Revision, version.Build)

	fmt.Println()

	address := ads.AddLocalConnection()

	variable := address.Symbols["ALARMS.WorkingAlarms"]
	variable.GetStringValue()

	// variable.GetStringValue()
	iface := variable.ParseNode()
	jsonObj, _ := json.Marshal(iface)
	fmt.Println(string(jsonObj))

	variable.AddNotification(4, uint32(time.Second), uint32(time.Second), sendJson)

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
	jsonEnc := symbol.ParseNode()
	jsonMarshal, _ := json.Marshal(jsonEnc)
	fmt.Println(string(jsonMarshal))
}
