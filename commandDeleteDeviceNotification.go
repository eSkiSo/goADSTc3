package ads

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/rs/zerolog/log"
)

// DeleteDeviceNotification does stuff
func (conn *Connection) DeleteDeviceNotification(handle uint32) {
	conn.waitGroup.Add(1)
	defer conn.waitGroup.Done()
	request := &bytes.Buffer{}
	type deleteNotificationCommandPacket struct {
		handle uint32
	}
	var content = deleteNotificationCommandPacket{
		handle,
	}
	binary.Write(request, binary.LittleEndian, content)
	// Try to send the request
	resp, err := conn.sendRequest(CommandIDDeleteDeviceNotification, request.Bytes())
	if err != nil {
		log.Info().
			Int("handle", int(handle)).
			Err(err).
			Msg("error deleting handle")
		return
	}

	// Check the result error code
	respBuffer := bytes.NewBuffer(resp)
	var adsError ReturnCode
	binary.Read(respBuffer, binary.LittleEndian, &adsError)
	if adsError > 0 {
		log.Info().
			Int("handle", int(handle)).
			Int("error", int(adsError)).
			Msg("error deleting handle")
		err = fmt.Errorf("Got ADS error number %d in DeleteDeviceNotification", adsError)
		return
	}
	delete(conn.activeNotifications, handle)
	log.Info().
		Int("handle", int(handle)).
		Msg("deleting handle")
	return
}
