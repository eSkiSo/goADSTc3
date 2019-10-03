package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/xilix-systems-llc/go-native-ads/v3/ads"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	client, _ := ads.AddRemoteConnection(ctx, "169.254.88.102.1.1", 851)
	go func() {
		for {
			select {
			case response := <-client.Notification:
				fmt.Printf("Value %s \n", response.Value)
			}
		}
	}()

	client.AddNotification("MAIN.i")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	<-c
	cancel()
	ads.CloseConnection()
	fmt.Println("closing")
	// sig is a ^C, handle it
	os.Exit(1)
}
