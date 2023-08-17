package clip

import (
	"time"

	"github.com/atotto/clipboard"
)

type CancelFunc func()

func NewStream(duration time.Duration) (chan string, CancelFunc) {
	channel := make(chan string)
	stopper := make(chan struct{})

	ticker := time.NewTicker(duration)
	go func() {
		prev := ""
		for {
			select {
			case <-stopper:
				close(channel)
				return
			case <-ticker.C:
				data, err := clipboard.ReadAll()
				if err != nil {
					continue
				}

				if prev == data {
					continue
				}
				channel <- data
				prev = data
			}
		}
	}()

	return channel, func() {
		close(stopper)
	}
}
