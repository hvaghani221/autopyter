package code

import (
	"io/fs"
	"time"

	"github.com/hvaghani221/autopyter/internal/clip"
)

type Code struct {
	clipStream chan string
	cancelFunc func()
	history    *History
	state      *State
	fs         fs.FS
	debug      bool
}

func NewCode(fs fs.FS, debug ...bool) *Code {
	clipStream, cancleFunc := clip.NewStream(time.Millisecond * 100)
	code := &Code{
		clipStream: clipStream,
		cancelFunc: cancleFunc,
		history:    NewHistory(),
		fs:         fs,
		state:      NewState(),
	}

	if len(debug) > 0 {
		code.debug = debug[0]
	}

	go code.listenStream()
	return code
}

func (c *Code) listenStream() {
	for data := range c.clipStream {
		c.history.Add(data)
	}
}

func (c *Code) Close() {
	c.cancelFunc()
	c.state.Close()
}
