package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.com/xilix-systems-llc/go-native-ads/v3/ads"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	client, _ := ads.AddRemoteConnection(ctx, "5.15.131.166.1.1", 851)
	go func() {
		for {
			select {
			case response := <-client.Update:
				fmt.Printf("Value %s \n", response)
			}
		}
	}()

	client.AddNotification("GVL.AllAlarms", ads.ADSTRANS_SERVERONCHA, 10*time.Millisecond, 100*time.Millisecond)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	<-c
	cancel()
	ads.Shutdown()
	fmt.Println("closing")
	// sig is a ^C, handle it
	os.Exit(1)
}
