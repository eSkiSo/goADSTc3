package ads

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

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
			binary.Read(data, binary.LittleEndian, &sample)
			content = make([]byte, sample.Size)
			data.Read(content)
			conn.activeNotificationLock.Lock()
			notification, ok := conn.activeNotifications[sample.Handle]
			update := symbolUpdate{
				data:      content,
				timestamp: time.Now(),
			}
			if ok {
				ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
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

			} else {
				err = fmt.Errorf("error finding callback for notification")
			}
			conn.activeNotificationLock.Unlock()
		}
	}
	return err
}
