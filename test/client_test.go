package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Ghytro/easytcp"
	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite

	ctx    context.Context
	cancel context.CancelFunc
}

func (s *ClientTestSuite) BeforeTest(_, _ string) {
	s.ctx, s.cancel = context.WithCancel(context.Background())
}

func (s *ClientTestSuite) AfterTest(_, _ string) {
	s.cancel()
}

func (s *ClientTestSuite) TestClientConnection() {
	server := prepareDefaultServer(s.T())

	go func() {
		server.Listen(s.ctx, port)
	}()

	time.Sleep(time.Millisecond * 500)

	client, err := easytcp.NewClient(s.ctx, easytcp.ClientConfig{
		Address:      port,
		MaxConns:     3,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		DialTimeout:  time.Second * 1,
	})
	if err != nil {
		s.FailNow(err.Error())
	}
	s.NoError(err)
	const workerAmount = 10
	var wg sync.WaitGroup
	wg.Add(workerAmount)
	for i := 0; i < workerAmount; i++ {
		i := i
		go func() {
			err := client.WithSession(func(conn easytcp.IConnection) error {
				fmt.Printf("client %d entered\n", i)
				n, err := conn.Write([]byte(stringPayload))
				if err != nil {
					return err
				}
				s.Equal(len(stringPayload), n)

				time.Sleep(time.Millisecond * 500)
				msg := make([]byte, len(stringPayload))
				n, err = conn.Read(msg)
				if err != nil {
					return err
				}
				s.Equal(len(stringPayload), n)
				s.Equal(stringPayload, string(msg))

				return nil
			})
			s.NoError(err)
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
