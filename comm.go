package ads

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/rs/zerolog/log"
)

func (conn *Connection) send(data []byte) (response []byte, err error) {
	conn.waitGroup.Add(1)
	defer conn.waitGroup.Done()

	ctx, cancel := context.WithCancel(conn.ctx)
	defer cancel()
	select {
	case <-conn.ctx.Done():
		return response, err
	case conn.sendChannel <- data:
	default:
		return
	}

	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			err = fmt.Errorf("Request aborted, deadline exceeded %w", ctx.Err())
			log.Error().
				Err(err).
				Msg("sendRequest aborted due to timeout")
		} else {
			err = fmt.Errorf("Request aborted, shutdown initiated %w", ctx.Err())
			log.Error().
				Err(err).
				Msg("sendRequest aborted due to shutdown")
		}
		return nil, err
	case response = <-conn.systemResponse:
		return response, nil
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
	conn.activeRequestLock.Lock()
	responseMap := conn.activeRequests[command]
	// First, request a new invoke id
	id := responseMap.id.Inc()
	// Create a channel for the response
	responseMap.response[id] = make(chan []byte)
	conn.activeRequestLock.Unlock()
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

	ctx, cancel := context.WithCancel(conn.ctx)
	defer cancel()
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Error().
				Msg("sendRequest aborted due to timeout")
		} else {
			log.Info().
				Msg("sendRequest aborted due to shutdown")
		}
		return nil, ctx.Err()
	case conn.sendChannel <- pack:
		break
	}
	ctx, cancel = context.WithCancel(conn.ctx)
	defer cancel()
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Error().
				Msg("sendRequest aborted due to timeout")
		} else {
			log.Info().
				Msg("sendRequest aborted due to shutdown")
		}
		return nil, ctx.Err()
	case response = <-responseMap.response[id]:
		return response, nil
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
			for {
				ctx, cancel := context.WithCancel(conn.ctx)
				defer cancel()
				select {
				case <-ctx.Done():
					log.Info().
						Msgf("exit listen")
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
								Err(err).
								Msg("EOF Error")
							//break readLoop
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
				//conn.activeRequestLock.Lock()
				if responseMap, ok := conn.activeRequests[header.Command]; ok {
					if response, ok := responseMap.response[header.InvokeID]; ok {
						ctx, cancel := context.WithCancel(conn.ctx)
						defer cancel()
						// Try to send the response to the waiting request function
						select {
						case <-ctx.Done():
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
							break
						}
					}
				} else {
					log.Debug().
						Bytes("data", buff.Bytes()).
						Uint32("invokeId", header.InvokeID).
						Msg("Got broadcast, invoke: ")
				}
				//conn.activeRequestLock.Unlock()
			}
		}
	}

}

func (conn *Connection) transmitWorker() {
	conn.waitGroup.Add(1)
	defer conn.waitGroup.Done()
	for {
		ctx, cancel := context.WithCancel(conn.ctx)
		defer cancel()
		select {
		case <-ctx.Done():
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
