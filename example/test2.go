package main

import (
	"bytes"
	"encoding/binary"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Logger = log.With().Caller().Logger()
}

type test struct {
	Name   uint32
	Blargh eeeks
}

type totalTest struct {
	test
	Data []byte
}

type eeeks uint16

func main2() {
	eeks := []byte{0x23, 0x24, 0x25}
	boop := test{Name: 16, Blargh: 16}
	buff := bytes.Buffer{}
	err := binary.Write(&buff, binary.LittleEndian, boop)
	log.Info().
		Interface("interface", boop).
		Hex("data", buff.Bytes()).
		Msg("Data1")
	err = binary.Write(&buff, binary.LittleEndian, eeks[:])
	log.Info().
		Interface("interface", boop).
		Bytes("data", buff.Bytes()).
		Msg("Data2")
	if err != nil {
		log.Error().
			Err(err).
			Msg("error")
	}
	log.Info().
		Interface("interface", boop).
		Bytes("data", buff.Bytes()).
		Msg("Data3")
}
