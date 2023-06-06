package easytcp

import (
	"context"
	"errors"
	"time"

	"github.com/Ghytro/easytcp/internal/common"
	"github.com/Ghytro/easytcp/internal/connection"
)

type Client struct {
	readTimeout  time.Duration
	writeTimeout time.Duration
	pool         *connection.Pool
}

type ClientConfig struct {
	Address      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	DialTimeout  time.Duration

	// MaxConns configurates maximum amount of connections
	// in the pool. If zero or less is given, the amount
	// of connections is unlimited
	MaxConns int
}

// setDefault sets all the unset fields to default values
func (c *ClientConfig) setDefault() {
	if c.ReadTimeout == 0 {
		c.ReadTimeout = DefaultClientConfig.ReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = DefaultClientConfig.WriteTimeout
	}
	if c.DialTimeout == 0 {
		c.DialTimeout = DefaultClientConfig.DialTimeout
	}
	if c.MaxConns <= 0 {
		c.MaxConns = DefaultClientConfig.MaxConns
	}
}

func (c *ClientConfig) Validate() error {
	if c.Address == "" {
		return common.WrapErr(errors.New("client config connection address is not set"))
	}
	return nil
}

var DefaultClientConfig = ClientConfig{
	ReadTimeout:  time.Second * 10,
	WriteTimeout: time.Second * 10,
	DialTimeout:  time.Second * 10,
	MaxConns:     10, // let's start with 10, then increase if needed
}

func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg.setDefault()
	if cfg.Address == "" {
		return nil, common.WrapErr(errors.New("client's remote address not specified"))
	}
	pool, err := connection.NewPool(ctx, cfg.Address, cfg.MaxConns, connection.ConnectionConfig{
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		DialTimeout:  cfg.DialTimeout,
	})
	if err != nil {
		return nil, err
	}
	return &Client{
		readTimeout:  cfg.ReadTimeout,
		writeTimeout: cfg.WriteTimeout,
		pool:         pool,
	}, nil
}

func (c *Client) WithSession(fn func(conn IConnection) error) error {
	conn, err := c.pool.Acquire()
	if err != nil {
		return err
	}
	defer c.pool.Release(conn)
	return fn(conn)
}

type IConnection interface {
	connection.IConnectionMixin
	connection.IConnectionReader
	connection.IConnectionWriter
}
