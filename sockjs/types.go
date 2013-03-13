package sockjs

/*
Cotains package internal types (not public)
*/

import (
	"errors"
	"io"
	"time"
)

// Error variable
var ErrConnectionClosed = errors.New("Connection closed.")

type context struct {
	Config
	HandlerFunc
	connections
}

type conn struct {
	context
	input_channel    chan []byte
	output_channel   chan []byte
	quit_channel     chan bool
	timeout          time.Duration
	httpTransactions chan *httpTransaction
}

func newConn(ctx *context) *conn {
	return &conn{
		input_channel:    make(chan []byte),
		output_channel:   make(chan []byte, 64),
		quit_channel:     make(chan bool),
		httpTransactions: make(chan *httpTransaction),
		timeout:          time.Second * 30,
		context:          *ctx,
	}
}

func (this *conn) ReadMessage() ([]byte, error) {
	select {
	case <-this.quit_channel:
		return []byte{}, io.EOF
	case val := <-this.input_channel:
		return val[1 : len(val)-1], nil
	}
	panic("unreachable")
}

func (this *conn) WriteMessage(val []byte) (count int, err error) {
	val2 := make([]byte, len(val))
	copy(val2, val)
	select {
	case this.output_channel <- val2:
	case <-time.After(this.timeout):
		return 0, ErrConnectionClosed
	case <-this.quit_channel:
		return 0, ErrConnectionClosed
	}
	return len(val), nil
}

func (this *conn) Close() (err error) {
	defer func() {
		if recover() != nil {
			err = ErrConnectionClosed
		}
	}()
	close(this.quit_channel)
	return
}

type connectionStateFn func(*conn) connectionStateFn

func (this *conn) run(cleanupFn func()) {
	for state := openConnectionState; state != nil; {
		state = state(this)
	}
	cleanupFn()
}
