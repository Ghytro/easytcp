package test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	suite.Suite

	ctx    context.Context
	cancel context.CancelFunc
}

func (s *ServerTestSuite) BeforeTest(_, _ string) {
	s.ctx, s.cancel = context.WithCancel(context.Background())
}

func (s *ServerTestSuite) AfterTest(_, _ string) {
	s.cancel()
}

func (s *ServerTestSuite) TestBasicPacket() {
	server := prepareDefaultServer(s.T())

	go func() {
		server.Listen(s.ctx, port)
	}()

	time.Sleep(time.Millisecond * 500)
	client, err := net.Dial("tcp", port)
	s.NoError(err)

	for i := 0; i < 5; i++ {
		n, err := client.Write([]byte(stringPayload))
		s.Equal(n, len(stringPayload))
		s.NoError(err)
		time.Sleep(time.Millisecond * 500)

		b := make([]byte, len(stringPayload))
		n, err = client.Read(b)
		s.Equal(len(stringPayload), n)
		s.NoError(err)
		s.Equal(stringPayload, string(b))
		time.Sleep(time.Millisecond * 500)
	}
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
