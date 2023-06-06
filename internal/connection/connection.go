package connection

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/Ghytro/easytcp/internal/common"
)

type IConnectionMixin interface {
	Dial() error
}

type IConnectionReader interface {
	IConnectionMixin
	io.Reader
	ReadContext(ctx context.Context, b []byte) (n int, err error)
}

type IConnectionWriter interface {
	IConnectionMixin
	io.Writer
	WriteContext(ctx context.Context, b []byte) (n int, err error)
}

type ConnectionConfig struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	DialTimeout  time.Duration
}

func (c *ConnectionConfig) setDefault() {
	if c.ReadTimeout == 0 {
		c.ReadTimeout = DefaultConnectionConfig.ReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = DefaultConnectionConfig.WriteTimeout
	}
	if c.DialTimeout == 0 {
		c.DialTimeout = DefaultConnectionConfig.DialTimeout
	}
}

var DefaultConnectionConfig = ConnectionConfig{
	ReadTimeout:  time.Second * 10,
	WriteTimeout: time.Second * 10,
	DialTimeout:  time.Second * 10,
}

// Connection is an extended structure for standard library net.Conn.
// It has additional functionality such as timeouts or reading with context.
// It's not recommended to use this structure directly in your code, because
// the functionality of client/server is enough. But the usage is still available
// only if you really need more of low-level socket control.
//
// THIS STRUCTURE IS NOT THREAD SAFE AND IS NOT MEANT FOR MULTITHREAD USAGE
type Connection struct {
	// underlying connection context with cancelation signal
	ctx context.Context

	// the connection itself
	conn net.Conn

	readTimeout  time.Duration
	writeTimeout time.Duration
	dialTimeout  time.Duration

	// If the connection was closed, this channel notifies the consumer.
	// The signal will be produced once, so this channel needs to be piped
	closeNotifier chan struct{}
	notifyOnce    sync.Once

	// bufferedByte determines if the client sent packets to server.
	// Is used in Connection.WaitForPacket
	bufferedByte *byte
}

func NewConnection(ctx context.Context, conn net.Conn, config ...ConnectionConfig) *Connection {
	if len(config) == 0 {
		config = append(config, DefaultConnectionConfig)
	}
	cfg := config[0]
	cfg.setDefault()
	return &Connection{
		ctx:           ctx,
		conn:          conn,
		readTimeout:   cfg.ReadTimeout,
		writeTimeout:  cfg.WriteTimeout,
		dialTimeout:   cfg.DialTimeout,
		closeNotifier: make(chan struct{}, 1),
	}
}

func (c *Connection) Dial() (err error) {
	var conn net.Conn
	err = common.WithTimeout(
		c,
		c.dialTimeout,
		func(tcpConn *Connection) error {
			conn, err = net.Dial("tcp", c.RemoteAddr())
			return err
		},
		func(tcpConn *Connection) error {
			return conn.Close()
		},
	)

	return nil
}

// WaitForPacket blocks until the packet arrives to socket.
// The only way to stop awaiting the incoming packet is to close the connection
func (c *Connection) WaitForPacket() error {
	if c.bufferedByte != nil {
		return nil
	}
	temp := make([]byte, 1)
	n, err := c.conn.Read(temp)
	if err != nil {
		return err
	}
	if n != 1 {
		return common.WrapErr(errors.New("wait byte was not received"))
	}
	c.bufferedByte = &temp[0]
	return nil
}

func (c *Connection) Read(b []byte) (n int, err error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.readTimeout)
	defer cancel()
	return c.ReadContext(ctx, b)
}

func (c *Connection) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

func (c *Connection) ReadContext(ctx context.Context, b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, errors.New("received zero length of reader buffer")
	}
	offset := 0
	// if the packet was awaited we need to store
	// temporary byte too
	if c.bufferedByte != nil {
		b[0] = *c.bufferedByte
		c.bufferedByte = nil
		offset = 1
		if len(b) == 1 {
			return 1, nil
		}
	}
	err = common.WithContext(
		ctx,
		c,
		func(c *Connection) error {
			n, err = c.conn.Read(b[offset:])
			n += offset
			return err
		},
		func(c *Connection) error {
			return c.Close()
		},
	)
	if err != nil {
		err := err.(*common.WithTimeoutError)
		if err.CleanupErr == nil {
			err.CleanupErr = c.Close()
		}
		return n, common.NestedCloseConnErr(err.Err, err.CleanupErr)
	}
	return
}

func (c *Connection) Write(b []byte) (n int, err error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.writeTimeout)
	defer cancel()
	return c.WriteContext(ctx, b)
}

func (c *Connection) WriteContext(ctx context.Context, b []byte) (n int, err error) {
	err = common.WithContext(
		ctx,
		c,
		func(c *Connection) error {
			n, err = c.conn.Write(b)
			return err
		},
		func(c *Connection) error {
			return c.Close()
		},
	)
	if err != nil {
		err := err.(*common.WithTimeoutError)
		if err.CleanupErr == nil {
			err.CleanupErr = c.Close()
		}
		return n, common.NestedCloseConnErr(err.Err, err.CleanupErr)
	}
	return
}

func (c *Connection) Close() error {
	var err error
	c.notifyOnce.Do(func() {
		c.closeNotifier <- struct{}{}
		err = c.conn.Close()
	})
	return err
}

func (c *Connection) CloseNotifier() <-chan struct{} {
	return c.closeNotifier
}
