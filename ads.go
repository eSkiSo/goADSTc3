package ads

import (
	"bytes"
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
			handle, err := conn.GetHandleByName(symbolName)
			if err != nil {
				return nil, err
			}
			localSymbol.Handle = handle
		}
		log.Trace().
			Interface("symbol", localSymbol).
			Msg("symbol got")
		return localSymbol, nil
	}
	err := fmt.Errorf("symbol does not exist")
	log.Error().
		Err(err).
		Str("symbol name", symbolName).
		Msg("error getting handle by name")
	return nil, err
}

func (conn *Connection) GetHandleByName(symbolName string) (handle uint32, err error) {
	resp, err := conn.WriteRead(uint32(GroupSymbolHandleByName), 0, 4, []byte(symbolName))
	if err != nil {
		log.Error().
			Err(err).
			Str("symbol name", symbolName).
			Msg("error getting handle by name")
		return 0, err
	}
	handle = binary.LittleEndian.Uint32(resp)
	return handle, err
}

func (conn *Connection) WriteToSymbol(symbolName string, value string) error {
	symbol, err := conn.GetSymbol(symbolName)
	if err != nil {
		log.Error().
			Err(err).
			Msg("error getting symbol")
		return err
	}
	data, err := symbol.writeToNode(value, 0, conn.datatypes)
	if err != nil {
		log.Error().
			Err(err).
			Msg("error during write to symbol")
		return err
	}
	conn.Write(uint32(GroupSymbolValueByHandle), symbol.Handle, data)
	return nil
}

func (conn *Connection) ReadFromSymbol(symbolName string) (string, error) {
	symbol, err := conn.GetSymbol(symbolName)
	if err != nil {
		log.Error().
			Err(err).
			Msg("error getting symbol")
		return "", err
	}
	data, err := conn.Read(uint32(GroupSymbolValueByHandle), symbol.Handle, symbol.Length)
	if err != nil {
		log.Error().
			Err(err).
			Msg("error during read symbol")
		return "", err
	}
	value, err := symbol.parse(data, 0)
	if err != nil {
		log.Error().
			Err(err).
			Msg("error during parse symbol")
		return "", err
	}
	symbol.Value = value
	return value, nil
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

func (conn *Connection) AddSymbolNotification(symbolName string, updateReceiver chan Update) error {
	symbol, err := conn.GetSymbol(symbolName)
	if err != nil {
		log.
			Error().
			Str("symbol", symbolName).
			Err(err).
			Msg("error getting symbol")
		return err
	}
	handle, err := conn.AddDeviceNotification(
		uint32(GroupSymbolValueByHandle),
		symbol.Handle,
		symbol.Length,
		TransModeServerOnChange,
		30*time.Millisecond,
		30*time.Millisecond)
	if err != nil {
		return err
	}
	update := conn.notificationHandler(symbol, updateReceiver)
	conn.activeNotifications[handle] = update
	return nil
}

type Update struct {
	Variable  string
	Value     string
	TimeStamp time.Time
}

/// Sample notification handler
func (conn *Connection) notificationHandler(symbol *Symbol, updateReceiver chan Update) chan symbolUpdate {
	update := make(chan symbolUpdate)
	go func() {
		conn.waitGroup.Add(1)
		defer conn.waitGroup.Done()
		for {
			select {
			case <-conn.ctx.Done():
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
				update := Update{
					Variable:  symbol.FullName,
					Value:     value,
					TimeStamp: receivedUpdate.timestamp,
				}
				select {
				case <-conn.ctx.Done():
					break
				case updateReceiver <- update:
				case <-time.After(250 * time.Millisecond):
				default:
					break
				}
			}
		}
	}()
	return update
}
