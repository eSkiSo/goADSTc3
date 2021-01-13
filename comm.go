package ads

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/rs/zerolog/log"
)

func (conn *Connection) send(data []byte) (response []byte, err error) {
	conn.waitGroup.Add(1)
	defer conn.waitGroup.Done()

	ctx, cancel := context.WithCancel(conn.ctx)
	defer cancel()
	select {
	case <-ctx.Done():
		return response, err
	case conn.sendChannel <- data:
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
	ctx, cancel = context.WithTimeout(conn.ctx, 1000*time.Millisecond)
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
}

func (conn *Connection) listen() <-chan []byte {
	c := make(chan []byte)
	go func() {
		defer close(c)
		defer conn.ctx.Done()
		reader := bufio.NewReader(conn.connection)
		buff := bytes.Buffer{}
		for {
			tcpHeader := amsTCPHeader{}
			data := make([]byte, binary.Size(tcpHeader))
			ctx, cancel := context.WithCancel(conn.ctx)
			defer cancel()
			select {
			case <-ctx.Done():
				log.Info().
					Msgf("exit listen")
				return
			default:
				_, err := io.ReadFull(reader, data)
				if err != nil {
					log.Info().
						Err(err).
						Msg("listen read error")
					return
				}
				break
			}
			buff.Write(data)
			err := binary.Read(&buff, binary.LittleEndian, &tcpHeader)
			if err != nil {
				log.Error().
					Err(err).
					Msg("error during header read")
				continue
			}
			data = make([]byte, tcpHeader.Length)
			ctx, cancel = context.WithCancel(conn.ctx)
			defer cancel()
			select {
			case <-ctx.Done():
				continue
			default:
				io.ReadFull(reader, data)
			}
			if tcpHeader.System > 0 {
				conn.systemResponse <- data
			} else {
				go conn.handleReceive(ctx, data)
			}
		}
	}()
	return c
}

func (conn *Connection) handleReceive(ctx context.Context, data []byte) {
	log.Trace().
		Msg("in read")
	buff := bytes.NewBuffer(data)
	header := amsHeader{}
	err := binary.Read(buff, binary.LittleEndian, &header)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Error parsing header")
		return
	}
	log.Trace().
		Interface("header", header).
		Msg("header info")

	// adsData := make([]byte, header.Length)
	// err = binary.Read(buff, binary.LittleEndian, &adsData)
	adsData := data[32:]
	if len(adsData) != int(header.Length) {
		log.Error().
			Err(err).
			Msg("Error parsing body")
		return
	}
	switch header.Command {
	case CommandIDDeviceNotification:
		err := conn.DeviceNotification(ctx, adsData)
		if err != nil {
			log.Error().
				Err(err).
				Msg("error")
		}
		break
	case CommandIDReadState:
		type readStateResponse struct {
			Error ReturnCode
			states
		}
		stateResponse := &readStateResponse{}
		buff := bytes.NewBuffer(adsData)
		err := binary.Read(buff, binary.LittleEndian, stateResponse)
		if err != nil {
			return
		}
		log.Info().
			Interface("AdsState", stateResponse.AdsState).
			Interface("DeviceState", stateResponse.DeviceState).
			Msg("response.ADSState")
		break
	default:
		// Check if the response channel exists and is open
		conn.activeRequestLock.Lock()
		if responseMap, ok := conn.activeRequests[header.Command]; ok {
			if response, ok := responseMap.response[header.InvokeID]; ok {
				ctx, cancel := context.WithCancel(ctx)
				defer cancel()
				// Try to send the response to the waiting request function
				select {
				case <-ctx.Done():
					log.Info().
						Uint32("id", header.InvokeID).
						Interface("command", header.Command).
						Msg("receive channel timed out")
					conn.activeRequestLock.Unlock()
					break
				case response <- adsData:
					log.Trace().
						Uint32("id", header.InvokeID).
						Interface("command", header.Command).
						Msgf("Successfully deliverd answer")
					break
					// default:
					// 	log.Trace().
					// 		Uint32("id", header.InvokeID).
					// 		Interface("command", header.Command).
					// 		Msgf("unable to send to getter")
					// 	break
				}

			} else {
				log.Error().
					Bytes("data", buff.Bytes()).
					Uint32("invokeId", header.InvokeID).
					Msg("Got broadcast, invoke: ")
			}

		} else {
			log.Error().
				Bytes("data", buff.Bytes()).
				Uint32("invokeId", header.InvokeID).
				Msg("Got broadcast, invoke: ")
		}
		conn.activeRequestLock.Unlock()
	}

}
func (conn *Connection) receiveWorker() {
	conn.waitGroup.Add(1)
	defer conn.waitGroup.Done()
	read := conn.listen()
	for {
		ctx, cancel := context.WithCancel(conn.ctx)
		defer cancel()
		select {
		case <-ctx.Done():
			log.Debug().
				Msg("Exit receiveWorker")
			return
		case data := <-read:
			log.Trace().
				Msg("in read")
			buff := bytes.NewBuffer(data)
			header := amsHeader{}
			err := binary.Read(buff, binary.LittleEndian, &header)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Error parsing header")
				conn.requestLock.Unlock()
				break
			}
			log.Trace().
				Interface("header", header).
				Msg("header info")
			adsData := make([]byte, header.Length)
			err = binary.Read(buff, binary.LittleEndian, &adsData)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Error parsing body")
				break
			}
			switch header.Command {
			case CommandIDDeviceNotification:
				err := conn.DeviceNotification(ctx, adsData)
				if err != nil {
					log.Error().
						Err(err).
						Msg("error")
				}
				break
			case CommandIDReadState:
				type readStateResponse struct {
					Error ReturnCode
					states
				}
				stateResponse := &readStateResponse{}
				buff := bytes.NewBuffer(adsData)
				err := binary.Read(buff, binary.LittleEndian, stateResponse)
				if err != nil {
					break
				}
				log.Info().
					Interface("AdsState", stateResponse.AdsState).
					Interface("DeviceState", stateResponse.DeviceState).
					Msg("response.ADSState")
				break
			default:
				// Check if the response channel exists and is open
				// conn.activeRequestLock.Lock()
				if responseMap, ok := conn.activeRequests[header.Command]; ok {
					if response, ok := responseMap.response[header.InvokeID]; ok {
						ctx, cancel := context.WithCancel(ctx)
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
							log.Trace().
								Uint32("id", header.InvokeID).
								Interface("command", header.Command).
								Msgf("unable to send to getter")
							break
						}

					} else {
						log.Error().
							Bytes("data", buff.Bytes()).
							Uint32("invokeId", header.InvokeID).
							Msg("Got broadcast, invoke: ")
					}
				} else {
					log.Error().
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
	writer := bufio.NewWriter(conn.connection)
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
			_, err := writer.Write(data)
			// _, err := conn.connection.Write(data)
			if err != nil {
				log.Error().
					Err(err).
					Msgf("Error sending data on conn")
			}
			writer.Flush()
		}
	}

}
