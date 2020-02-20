package main

import (
	"flag"
	"fmt"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"

	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	ads "gitlab.com/xilix-systems-llc/go-native-ads/v4"
)

var WaitGroup sync.WaitGroup

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Logger = log.With().
		Caller().
		Logger()
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var debug = flag.Bool("debug", false, "print debugging messages.")
var ip = flag.String("ip", "10.0.1.235", "the address to the AMS router")

// ip := flag.String("ip", "127.0.0.1", "the address to the AMS router")
var netid = flag.String("netid", "10.0.2.15.1.1", "AMS NetID of the target")
var port = flag.Int("port", 48898, "AMS Port of the target")

// localNetid := flag.String("localNetId", "127.0.0.1.1.1", "AMS NetID of the target")
var localNetid = flag.String("localNetId", "10.0.1.245.1.1", "AMS NetID of the target")
var localPort = flag.Int("localPort", 10010, "AMS Port of the target")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal().
				Msg("fatal error")
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	type ADSSymbolUploadInfo struct {
		SymbolCount    uint32
		SymbolLength   uint32
		DataTypeCount  uint32
		DataTypeLength uint32
		ExtraCount     uint32
		ExtraLength    uint32
	}

	// Flags

	fmt.Println(*debug, *ip, *netid, *port)

	// Startup the connection

	connection, err := ads.NewConnection(*ip, *netid, *port, *localNetid, *localPort)
	connection.Connect(true)
	defer connection.Close() // Close the connection when we are done
	if err != nil {
		log.Error().
			Err(err).
			Msg("error")
		return
	}

	// Check what device are we connected to
	data, err := connection.ReadState()
	if err != nil {
		log.Error().
			Err(err).
			Msg("error")
		return
	}
	log.Info().
		Interface("adsState", data).
		Msg("Successfully conncected./test	")
	symbol, err := connection.GetSymbol("MAIN.I")
	if err != nil {
		return
	}
	// stringSymbol, err := connection.GetSymbol("MAIN.b")
	connection.WriteToSymbol("MAIN.b", "Say wut2!")
	log.Info().
		Interface("symbol", symbol).
		Msg("This is MAIN.I")
	start := time.Now()
	wg := sync.WaitGroup{}
	for i := 0; i < 20000; i++ {
		connection.WriteToSymbol("MAIN.g", strconv.Itoa(i))

		// if i%100 == 0 {
		// 	connection.WriteToSymbol("MAIN.b", "test"+strconv.Itoa(i))
		// 	state, _ := connection.ReadState()
		// 	log.Info().
		// 		Interface("state", state).
		// 		Msg("got state")
		// }
	}
	log.Info().
		Msg("I'm waiting for those stupid tasks to end")
	wg.Wait()
	log.Info().
		Dur("time to send", time.Since(start)).
		Msg("time to send 20k")
	connection.WriteToSymbol("MAIN.I", "0")
	log.Info().
		Interface("symbol", symbol).
		Msg("Final Value")
	value := connection.ReadFromSymbol("MAIN.I")
	log.Info().
		Interface("symbol", symbol).
		Str("value", value).
		Msg("Final Value")
	connection.AddSymbolNotification("MAIN.I")
	time.Sleep(5000 * time.Millisecond)
	connection.WriteToSymbol("MAIN.b", "Say wut3!")
	log.Info().
		Interface("symbol", symbol).
		Msg("This is MAIN.I")
	connection.WriteToSymbol("MAIN.I", "0")
	log.Info().
		Interface("symbol", symbol).
		Msg("Final Value")
}
