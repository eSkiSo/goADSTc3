package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	ads "gitlab.com/xilix-systems-llc/go-native-ads/v2/ads"
)

func main() {

	address, _ := ads.AddLocalConnection()

	variable, _ := address.Symbols.Load("GVL.TakePicture")

	variable.(*ads.ADSSymbol).AddNotification(4, time.Millisecond*100, time.Millisecond*100, sendJSON)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	<-c
	ads.CloseAllConnections()
	fmt.Println("closing")
	// sig is a ^C, handle it
	os.Exit(1)
}

func sendJSON(symbol ads.ADSSymbol) {
	// fmt.Println("Callback", symbol.Name, symbol.Value)
	jsonReturn, err := symbol.GetJSON(true)
	if err == nil {
		fmt.Println(string(jsonReturn))
	} else {
		fmt.Println(err)
	}
}
