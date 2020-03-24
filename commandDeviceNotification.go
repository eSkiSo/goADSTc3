package ads

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

const windowsTick int64 = 10000000
const secToUnixEpoch int64 = 11644473600

// DeviceNotification - ADS command id: 8
func (conn *Connection) DeviceNotification(ctx context.Context, in []byte) error {
	conn.waitGroup.Add(1)
	defer conn.waitGroup.Done()
	type NotificationStream struct {
		Length uint32
		Stamps uint32
	}
	type StampHeader struct {
		Timestamp uint64
		Samples   uint32
	}
	type NotificationSample struct {
		Handle uint32
		Size   uint32
	}

	var stream NotificationStream
	var header StampHeader
	var sample NotificationSample
	var content []byte

	data := bytes.NewBuffer(in)

	// Read stream header

	err := binary.Read(data, binary.LittleEndian, &stream)
	if err != nil {
		return fmt.Errorf("unable to read notification %v", err)
	}
	for i := uint32(0); i < stream.Stamps; i++ {
		// Read stamp header
		binary.Read(data, binary.LittleEndian, &header)

		for j := uint32(0); j < header.Samples; j++ {
			err := binary.Read(data, binary.LittleEndian, &sample)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Error during notification read")
				break
			}
			content = make([]byte, sample.Size)

			data.Read(content)
			conn.activeNotificationLock.Lock()
			notification, ok := conn.activeNotifications[sample.Handle]
			if !ok {
				log.Error().
					Msg("Can't find notification handle")

				conn.DeleteDeviceNotification(sample.Handle)
				conn.activeNotificationLock.Unlock()
				continue
			}
			timeStamp := int64(header.Timestamp)/windowsTick - secToUnixEpoch
			notificationTime := time.Unix(timeStamp, int64(header.Timestamp)%(windowsTick)*100)
			update := symbolUpdate{
				data:      content,
				timestamp: notificationTime,
			}
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			// Try to send the response to the waiting request function
			select {
			case <-ctx.Done():
				break
			case notification <- update:
				log.Debug().
					Msgf("Successfully delivered notification for handle %d", sample.Handle)
				break
			}
			conn.activeNotificationLock.Unlock()
		}
	}
	return err
}
