package ads

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/rs/zerolog/log"
)

// Write - ADS command id: 3
func (conn *Connection) Write(group uint32, offset uint32, data []byte) {
	conn.waitGroup.Add(1)
	defer conn.waitGroup.Done()
	type writeCommandPacket struct {
		Group  uint32
		Offset uint32
		Length uint32
	}
	request := new(bytes.Buffer)
	writeRequest := writeCommandPacket{
		group,
		offset,
		uint32(len(data)),
	}

	err := binary.Write(request, binary.LittleEndian, writeRequest)
	binary.Write(request, binary.LittleEndian, data)
	if err != nil {
		log.Error().
			Msgf("binary.Write failed: %s", err)
	}

	// Try to send the request
	resp, err := conn.sendRequest(CommandIDWrite, request.Bytes())
	if err != nil {
		log.Error().
			Err(err).
			Msg("error during send request for write")
		return
	}
	respBuffer := bytes.NewBuffer(resp)
	var respCode ReturnCode
	// Check the result error code
	err = binary.Read(respBuffer, binary.LittleEndian, &respCode)
	if respCode > 0 {
		log.Error().
			Err(err).
			Msg("error during write")
		err = fmt.Errorf("Got ADS error number %v in Write", respCode)
		return
	}

	return
}
