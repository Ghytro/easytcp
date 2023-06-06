package easytcp

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"sync"

	"github.com/Ghytro/easytcp/internal/common"
	"github.com/Ghytro/easytcp/internal/connection"
)

// ServerContext provides high-level control over the tcp connection
// with access to underlying context and storing connection-scoped
// values. The underlying context is cancelled whenever the client
// connection is closed
type ServerContext struct {
	ctx        context.Context
	server     *Server
	handlerIdx int
	vals       map[string]interface{}
	valMutex   *sync.Mutex
	conn       *connection.Connection
	resp       *bytes.Buffer
}

func (ctx *ServerContext) Context() context.Context {
	return ctx.ctx
}

func (ctx *ServerContext) Set(key string, value interface{}) {
	ctx.valMutex.Lock()
	defer ctx.valMutex.Unlock()
	ctx.vals[key] = value
}

// Value retreives value from context by it's key. Returns val if the value is not present,
// otherwise return nil
func (ctx *ServerContext) Get(key string, val ...interface{}) interface{} {
	ctx.valMutex.Lock()
	defer ctx.valMutex.Unlock()
	result, ok := ctx.vals[key]
	if !ok {
		if len(val) != 0 {
			return val[0]
		}
		return nil
	}
	return result
}

func (ctx *ServerContext) Delete(key string) {
	ctx.valMutex.Lock()
	defer ctx.valMutex.Unlock()
	delete(ctx.vals, key)
}

func (ctx *ServerContext) WriteBuf(b []byte) (int, error) {
	return ctx.resp.Write(b)
}

func (ctx *ServerContext) SendBuf() (int, error) {
	defer ctx.resp.Reset()
	return ctx.SendBinary(ctx.resp.Bytes())

}

func (ctx *ServerContext) Read(b []byte) (int, error) {
	return ctx.conn.Read(b)
}

func (ctx *ServerContext) WaitForPacket() error {
	return ctx.conn.WaitForPacket()
}

func (ctx *ServerContext) Next() error {
	if ctx.handlerIdx == len(ctx.server.handlers) {
		return nil
	}
	ctx.handlerIdx++
	return ctx.server.handlers[ctx.handlerIdx-1](ctx)
}

// SendBinary send passed byte slice to client
func (ctx *ServerContext) SendBinary(b []byte) (int, error) {
	return ctx.conn.Write(b)
}

// Send send data to client that can be either byte slice or encoding.BinaryMarshaller.
// Otherwise data will be json encoded and sent to client
func (ctx *ServerContext) Send(data interface{}) error {
	if data == nil {
		return nil
	}
	var (
		b   []byte
		err error
	)

	switch t := data.(type) {
	case []byte:
		b = t

	case string:
		b = common.Str2B(t)

	case *string:
		if t == nil {
			return nil
		}
		b = common.Str2B(*t)

	case encoding.BinaryMarshaler:
		b, err = t.MarshalBinary()

	default:
		b, err = json.Marshal(t)
	}
	if err != nil {
		return err
	}

	_, err = ctx.SendBinary(b)
	return err
}

func (ctx *ServerContext) RemoteAddr() string {
	return ctx.conn.RemoteAddr()
}
