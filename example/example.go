package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.com/xilix-systems-llc/go-native-ads/ads/v3"
)

func main() {
	lastTimes := make(map[string]time.Time)
	address, _ := ads.AddRemoteConnection("10.0.1.158.1.1", 851)
	go func() {
		for {
			select {
			case response := <-address.Notification:
				lastTime, ok := lastTimes[response.Variable]
				if !ok {
					lastTime = response.TimeStamp
					lastTimes[response.Variable] = lastTime
				}
				//fmt.Printf("Value %s Timespan %v\n", response.Value, lastTime)
				lastTimes[response.Variable] = response.TimeStamp
				fmt.Printf("Value %s Timespan %v\n", response.Value, lastTime)
			}
		}
	}()
	// go func() {
	// 	for {
	// 		// address.Read <- "MAIN.i"
	// 		// <-address.ReadResponse
	// 	}
	// }()
	// go func() {
	// 	for {
	// 		// address.Write <- ads.WriteStruct{Variable: "MAIN.i.i", Value: "0"}
	// 		// time.Sleep(time.Millisecond * 500)
	// 	}
	// }()
	// go func() {
	// 	blargh := 1
	// 	for {
	// 		blargh *= 25
	// 		//fmt.Printf("writing blargh: %d\n", blargh)
	// 		// address.Write <- ads.WriteStruct{Variable: "MAIN.i.c", Value: strconv.Itoa(blargh)}
	// 		if blargh > math.MaxInt8 {
	// 			blargh = 1
	// 		}
	// 		time.Sleep(time.Millisecond * 200)
	// 	}
	// }()
	address.AddNotification <- "GVL.AllAlarms"
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
