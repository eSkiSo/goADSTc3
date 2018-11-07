package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/xilix-systems-llc/go-native-ads/v2/ads"
)

func main() {

	address, _ := ads.AddLocalConnection()
	fmt.Print("here1")
	ads.RouterNotification = routerNotificatin
	fmt.Print("here2")
	ads.RegisterRouterNotification(routerNotificatin)
	fmt.Print("here3")
	// variable, _ := address.Symbols.Load("GVL.TakePicture")
	fmt.Print("here")
	a, b, _ := address.AdsSyncReadStateReq()
	fmt.Printf("a %f b: %f", a, b)
	// variable.(*ads.ADSSymbol).AddNotification(4, time.Millisecond*100, time.Millisecond*100, sendJSON)
	fmt.Print("again")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	<-c
	ads.CloseAllConnections()
	ads.UnregisterRouterNotification()
	fmt.Println("closing")
	// sig is a ^C, handle it
	os.Exit(1)
}

func routerNotificatin(response int) {
	fmt.Print(response)
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
