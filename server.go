package easytcp

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net"
	"time"

	"github.com/Ghytro/easytcp/internal/common"
	"github.com/Ghytro/easytcp/internal/connection"
)

type ServerHandler func(ctx *ServerContext) error

type ErrHandler func(ctx *ServerContext, err error) error

type ServerConfig struct {
	ReadTimeout, WriteTimeout time.Duration
}

func (c *ServerConfig) setDefault() {
	if c.ReadTimeout == 0 {
		c.ReadTimeout = DefaultServerConfig.ReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = DefaultServerConfig.WriteTimeout
	}
}

var DefaultServerConfig = ServerConfig{
	ReadTimeout:  time.Second * 10,
	WriteTimeout: time.Second * 10,
}

func DefaultErrorHandler(ctx *ServerContext, err error) error {
	log.Print(err)
	return nil
}

type Server struct {
	// handlers proceed the incoming tcp stream and put additional data in context
	handlers   []ServerHandler
	onConnect  ServerHandler
	errHandler ErrHandler

	// Timeout for an unmarshaller to proceed the incoming byte stream. This timeout
	// is only for network, so don't confuse it with your program's additional runtime delay
	unmarshallerTimeout time.Duration

	// Timeout to write a response to client. This timeout is only for network, so don't confuse
	// it with your program's additional runtime delay
	responseTimeout time.Duration
}

func NewServer(config ...ServerConfig) *Server {
	if len(config) == 0 {
		config = append(config, DefaultServerConfig)
	}
	cfg := config[0]
	cfg.setDefault()
	return &Server{
		unmarshallerTimeout: cfg.ReadTimeout,
		responseTimeout:     cfg.WriteTimeout,
	}
}

// Register adds a handler that proceeds the incoming packets or connections
func (s *Server) Register(fn ServerHandler) {
	s.handlers = append(s.handlers, fn)
}

// OnConnect is called when the new client connets to server
func (s *Server) OnConnect(fn ServerHandler) {
	s.onConnect = fn
}

func (s *Server) ErrorHandler(fn ErrHandler) {
	s.errHandler = fn
}

func (s *Server) handleErr(ctx *ServerContext, err error) error {
	if err == nil {
		return nil
	}
	if s.errHandler != nil {
		return s.errHandler(ctx, err)
	}
	return DefaultErrorHandler(ctx, err)
}

// Listen starts listening tcp connection via net.Listen. You can pass the
// additional context to stop listening when it's done. The method is blocking
// until the error occurs or context will be done and returns an error that explains
// why the connection was closed: was that a context, or some kind of internal error
func (s *Server) Listen(ctx context.Context, addr string) error {
	if err := s.validateBeforeListen(); err != nil {
		return common.WrapErr(err)
	}

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			if conn != nil {
				err = common.NestedCloseConnErr(err, conn.Close())
			}
			log.Print(common.WrapErr(err))
			continue
		}

		if conn.RemoteAddr().Network() != "tcp" {
			// todo: logger
			// err := errors.New("peer connected not via tcp, closing connection")
			// err = nestedCloseConnErr(err, conn.Close())
			if err := conn.Close(); err != nil {
				log.Print(err)
				continue
			}
			continue
		}

		go s.connHandler(conn)
	}
}

func (s *Server) connHandler(conn net.Conn) {
	// parent context notifies about closed connection
	parentCtx, notifyClosed := context.WithCancel(context.Background())
	tcpConn := connection.NewConnection(
		parentCtx,
		conn,

		connection.ConnectionConfig{
			ReadTimeout:  s.unmarshallerTimeout,
			WriteTimeout: s.responseTimeout,
		},
	)

	go func() {
		<-tcpConn.CloseNotifier()
		notifyClosed()
	}()

	defer tcpConn.Close()

	sCtx := &ServerContext{
		ctx:        parentCtx,
		server:     s,
		vals:       map[string]interface{}{},
		handlerIdx: 0,
		conn:       tcpConn,
		resp:       new(bytes.Buffer),
	}
	if s.onConnect != nil {
		if err := s.onConnect(sCtx); err != nil {
			s.handleErr(sCtx, err)
			return
		}
	}
	for {
		// execute all the attached handlers
		sCtx.handlerIdx = 0
		if err := sCtx.Next(); err != nil {
			s.handleErr(sCtx, err)
			return
		}

		// if the user has written the response, send in to socket
		if sCtx.resp.Len() != 0 {
			n, err := sCtx.SendBuf()
			if err != nil {
				err := common.WrapErr(common.NestedCloseConnErr(err, tcpConn.Close()))
				s.handleErr(sCtx, err)
				return
			}
			if n != sCtx.resp.Len() {
				err := errors.New("not all the bytes were written to response, connection closed")
				err = common.WrapErr(common.NestedCloseConnErr(err, tcpConn.Close()))
				s.handleErr(sCtx, err)
				return
			}
		}
	}
}

// validateBeforeListen check if all the fields are valid
// before launching the server
func (s *Server) validateBeforeListen() error {
	if len(s.handlers) == 0 {
		return errors.New("the handler cannot be nil, all the packets will be ignored")
	}
	return nil
}
