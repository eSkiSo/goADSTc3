package ads

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

func (conn *Connection) GetSymbol(symbolName string) (*Symbol, error) {
	conn.symbolLock.Lock()
	defer conn.symbolLock.Unlock()
	localSymbol, ok := conn.symbols[symbolName]
	if ok {
		if localSymbol.Handle == 0 {
			localSymbol.Handle = conn.GetHandleByName(symbolName)
		}
		log.Trace().
			Interface("symbol", localSymbol).
			Msg("symbol got")
		return &localSymbol, nil
	}
	err := fmt.Errorf("symbol does not exist")
	log.Error().
		Err(err).
		Msg("error getting handle by name")
	return nil, err
}

func (conn *Connection) GetHandleByName(symbolName string) (handle uint32) {
	resp, err := conn.WriteRead(uint32(GroupSymbolHandleByName), 0, 4, []byte(symbolName))
	if err != nil {
		log.Error().
			Err(err).
			Msg("error getting handle by name")
		return
	}
	handle = binary.LittleEndian.Uint32(resp)
	return
}

func (conn *Connection) WriteToSymbol(symbolName string, value string) {
	symbol, err := conn.GetSymbol(symbolName)
	if err != nil {
		log.Error().
			Err(err).
			Msg("error getting symbol")
		return
	}
	data, err := symbol.writeToNode(value, 0, conn.datatypes)
	if err != nil {
		log.Error().
			Err(err).
			Msg("error during write to symbol")
	}
	conn.Write(uint32(GroupSymbolValueByHandle), symbol.Handle, data)
}

func (conn *Connection) ReadFromSymbol(symbolName string) string {
	symbol, err := conn.GetSymbol(symbolName)
	if err != nil {
		log.Error().
			Err(err).
			Msg("error getting symbol")
	}
	data, err := conn.Read(uint32(GroupSymbolValueByHandle), symbol.Handle, symbol.Length)
	if err != nil {
		log.Error().
			Err(err).
			Msg("error during read symbol")
		return ""
	}
	value, err := symbol.parse(data, 0)
	if err != nil {
		log.Error().
			Err(err).
			Msg("error during parse symbol")
		return ""
	}
	symbol.Value = value
	return value
}

func (conn *Connection) GetSymbolUploadInfo() (uploadInfo SymbolUploadInfo, err error) {
	res, err := conn.Read(uint32(GroupSymbolUploadInfo2), 0, 24) //UploadSymbolInfo;
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Bad Bad Bad")
		return
	}
	buff := bytes.NewBuffer(res)
	binary.Read(buff, binary.LittleEndian, &uploadInfo)
	return
}

func (conn *Connection) GetUploadSymbolInfoSymbols(length uint32) (data []byte, err error) {
	res, err := conn.Read(uint32(GroupSymbolUpload), 0, length) //UploadSymbolInfo;
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Bad Bad Bad")
		return nil, err
	}
	return res, nil
}

func (conn *Connection) GetUploadSymbolInfoDataTypes(length uint32) (data []byte, err error) {
	data, err = conn.Read(
		uint32(GroupSymbolDataTypeUpload),
		0x0,
		length)
	if err != nil {
		return nil, fmt.Errorf("error doing DT UPLOAD %d", err)
	}
	return data, nil
}

func (conn *Connection) AddSymbolNotification(symbolName string) {
	symbol, err := conn.GetSymbol(symbolName)
	if err != nil {
		return
	}
	handle, err := conn.AddDeviceNotification(
		uint32(GroupSymbolValueByHandle),
		symbol.Handle,
		symbol.Length,
		TransModeServerOnChange,
		50*time.Millisecond,
		50*time.Millisecond)
	update := conn.notificationHandler(symbol)
	conn.activeNotifications[handle] = update
	return
}

func (conn *Connection) notificationHandler(symbol *Symbol) chan symbolUpdate {
	update := make(chan symbolUpdate)
	go func() {
		conn.waitGroup.Add(1)
		defer conn.waitGroup.Done()
		ctx, cancel := context.WithCancel(conn.ctx)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			case receivedUpdate := <-update:
				value, err := symbol.parse(receivedUpdate.data, 0)
				if err != nil {
					log.Error().
						Err(err).
						Msg("error during parse of notification")
					break
				}
				symbol.Value = value
				log.Trace().
					Str("update", symbol.Value).
					Msgf("update received")
				break
			}
		}
	}()
	return update
}
