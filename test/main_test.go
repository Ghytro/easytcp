package test

import (
	"os"
	"testing"
	"time"

	"github.com/Ghytro/easytcp"
	"github.com/stretchr/testify/require"
)

const (
	stringPayload = "abacaabaca"
	port          = ":9876"
)

func prepareDefaultServer(t *testing.T) *easytcp.Server {
	server := easytcp.NewServer(easytcp.ServerConfig{
		ReadTimeout:  time.Second * 2,
		WriteTimeout: time.Second * 2,
	})
	server.Register(func(ctx *easytcp.ServerContext) error {
		firstPart := make([]byte, 5)
		n, err := ctx.Read(firstPart)
		if err != nil {
			return err
		}
		require.Equal(t, 5, n)
		require.Equal(t, stringPayload[:5], string(firstPart))
		return ctx.Next()
	})
	server.Register(func(ctx *easytcp.ServerContext) error {
		secondPart := make([]byte, 5)
		n, err := ctx.Read(secondPart)
		if err != nil {
			return err
		}
		require.Equal(t, 5, n)
		require.Equal(t, stringPayload[5:], string(secondPart))
		return ctx.Send(stringPayload)
	})
	return server
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
