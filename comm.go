package ads

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"time"

	"github.com/rs/zerolog/log"
)

func (conn *Connection) send(data []byte) (response []byte, err error) {
	conn.waitGroup.Add(1)
	defer conn.waitGroup.Done()
	select {
	case <-conn.ctx.Done():
		return response, err
	case conn.sendChannel <- data:
	default:
		return
	}
	select {
	case <-conn.ctx.Done():
		return nil, err
	case <-time.After(250 * time.Millisecond):
		break
	case response = <-conn.systemResponse:
		break
	}
	return
}

func (conn *Connection) sendRequest(command CommandID, data []byte) (response []byte, err error) {
	conn.waitGroup.Add(1)
	defer conn.waitGroup.Done()
	if conn == nil {
		log.Error().
			Msg("Failed to encode header, connection is nil pointer")
		return
	}
	// First, request a new invoke id
	conn.requestLock.Lock()
	defer conn.requestLock.Unlock()
	responseMap := conn.activeRequests[command]
	// Create a channel for the response
	id := responseMap.id.Inc()
	responseMap.response[id] = make(chan []byte)
	log.Trace().
		Interface("command", command).
		Bytes("data", data).
		Uint32("id", id).
		Msg("encoding packet")

	pack, err := conn.encode(command, data, id)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Error during sendrequest encode")
		return nil, err
	}

	select {
	case <-conn.ctx.Done():
		break
	case conn.sendChannel <- pack:
		break
	case <-time.After(250 * time.Millisecond):
		return
	}

	select {
	case <-conn.ctx.Done():
		return nil, conn.ctx.Err()
	case response = <-responseMap.response[id]:
		return response, nil
	case <-time.After(250 * time.Millisecond):
	}
	return
}

func listen(conn *Connection) <-chan []byte {
	c := make(chan []byte)
	go func() {
		buff := &bytes.Buffer{}
		tmp := make([]byte, 256)
		for {
		readLoop:
			for { // using small tmo buffer for demonstrating
				select {
				case <-conn.ctx.Done():
					return
				default:
					if buff.Len() > 6 {
						break readLoop
					}
					n, err := conn.connection.Read(tmp)
					buff.Write(tmp[:n])
					if err != nil {
						log.Error().
							Err(err).
							Msg("error during tcp read")
						var timeoutError net.Error
						if errors.As(err, &timeoutError) {
							if timeoutError.Timeout() {
								log.Error().
									Msg("timeout error")
								conn.ReConnect()
							}
						}
						if errors.Is(err, io.EOF) {
							log.Error().
								Msg("eof error")
							conn.ReConnect()
						}
						break
					}
				}
			}

			tcpHeader := amsTCPHeader{}
			err := binary.Read(buff, binary.LittleEndian, &tcpHeader)
			if err != nil {
				log.Error().
					Err(err).
					Msg("error during header read")
			}
		bodyLoop:
			for { // using small tmo buffer for demonstrating
				select {
				case <-conn.ctx.Done():
					return
				default:
					if buff.Len() >= int(tcpHeader.Length) {
						break bodyLoop
					}
					n, err := conn.connection.Read(tmp)
					buff.Write(tmp[:n])
					if err != nil {
						log.Error().
							Msg("Error during read")
						break bodyLoop
					}
				}
			}
			data := make([]byte, tcpHeader.Length)
			err = binary.Read(buff, binary.LittleEndian, &data)
			if err != nil {
				log.Error().
					Err(err).
					Msg("read error")
			} else {
				log.Debug().
					Int("buffer length", buff.Len()).
					Uint32("header length", tcpHeader.Length).
					Msg("TCPHeader")
			}
			var receiveChan chan []byte
			if tcpHeader.System > 0 {
				receiveChan = conn.systemResponse
			} else {
				receiveChan = c
			}
			select {
			case <-time.After(250 * time.Millisecond):
			case receiveChan <- data:
			}

		}
	}()
	return c
}

func (conn *Connection) receiveWorker() {
	conn.waitGroup.Add(1)
	defer conn.waitGroup.Done()
	read := listen(conn)
	for {
		select {
		case <-conn.ctx.Done():
			log.Debug().
				Msg("Exit receiveWorker")
			return
		case data := <-read:
			log.Trace().
				Msg("in read")
			buff := bytes.NewBuffer(data)
			header := amsHeader{}
			binary.Read(buff, binary.LittleEndian, &header)
			log.Trace().
				Interface("header", header).
				Msg("header info")
			adsData := make([]byte, header.Length)
			binary.Read(buff, binary.LittleEndian, &adsData)
			switch header.Command {
			case CommandIDDeviceNotification:
				conn.DeviceNotification(conn.ctx, adsData)
				break
			case CommandIDReadState:
				type readStateResponse struct {
					Error ReturnCode
					states
				}
				stateResponse := &readStateResponse{}
				buff := bytes.NewBuffer(adsData)
				binary.Read(buff, binary.LittleEndian, stateResponse)
				log.Info().
					Interface("AdsState", stateResponse.AdsState).
					Interface("DeviceState", stateResponse.DeviceState).
					Msg("response.ADSState")
				fallthrough
			default:
				// Check if the response channel exists and is open
				if responseMap, ok := conn.activeRequests[header.Command]; ok {
					if response, ok := responseMap.response[header.InvokeID]; ok {
						// Try to send the response to the waiting request function
						select {
						case <-conn.ctx.Done():
							log.Info().
								Uint32("id", header.InvokeID).
								Interface("command", header.Command).
								Msg("receive channel timed out")
							break
						case response <- adsData:
							log.Trace().
								Uint32("id", header.InvokeID).
								Interface("command", header.Command).
								Msgf("Successfully deliverd answer")
							break
						default:
							log.Info().
								Uint32("id", header.InvokeID).
								Interface("command", header.Command).
								Msg("receive channel closed")
							break
						}
					}
				} else {
					log.Debug().
						Bytes("data", buff.Bytes()).
						Uint32("invokeId", header.InvokeID).
						Msg("Got broadcast, invoke: ")
				}
			}
		}
	}

}

func (conn *Connection) transmitWorker() {
	conn.waitGroup.Add(1)
	defer conn.waitGroup.Done()
	for {
		select {
		case <-conn.ctx.Done():
			log.Debug().
				Msg("Exit transmitWorker")
			return
		case data := <-conn.sendChannel:
			log.Trace().
				Msgf("Sending %d bytes", len(data))
			conn.connection.Write(data)
		}
	}

}
