package connection

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"

	"github.com/Ghytro/easytcp/internal/algo"
	"github.com/Ghytro/easytcp/internal/common"
	"golang.org/x/sync/semaphore"
)

type Pool struct {
	connCfg      ConnectionConfig
	address      string
	pool         []*poolEntry
	poolLock     *sync.Mutex
	clientWaiter *semaphore.Weighted
	ctx          context.Context
	maxSize      int
}

func NewPool(ctx context.Context, address string, size int, connCfg ...ConnectionConfig) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("it's strongly recommended to not create pool of non-fixed size")
	}
	if len(connCfg) == 0 {
		connCfg = append(connCfg, DefaultConnectionConfig)
	}

	result := &Pool{
		maxSize:      size,
		address:      address,
		connCfg:      connCfg[0],
		poolLock:     &sync.Mutex{},
		clientWaiter: semaphore.NewWeighted(int64(size)),
		ctx:          ctx,
	}
	entries := make([]*poolEntry, size)
	for i := 0; i < size; i++ {
		conn, err := net.Dial("tcp", address)
		if err != nil {
			return nil, err
		}
		tcpConn := NewConnection(ctx, conn, connCfg[0])
		entry := &poolEntry{
			conn:     tcpConn,
			acquired: 0,
		}
		entries[i] = entry
	}
	result.pool = entries
	return result, nil
}

func (p *Pool) Acquire() (*Connection, error) {
	if err := p.clientWaiter.Acquire(p.ctx, 1); err != nil {
		return nil, err
	}
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	entry, ok := algo.Find(p.pool, func(entry *poolEntry) bool {
		return atomic.CompareAndSwapInt32(&entry.acquired, 0, 1)
	})
	if !ok {
		return nil, errors.New("there are no available connections, try again later")
	}
	return entry.conn, nil
}

func (p *Pool) Release(conn *Connection) error {
	if conn == nil {
		return common.WrapErr(errors.New("cannot release an empty connection"))
	}
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	entry, ok := algo.Find(p.pool, func(entry *poolEntry) bool {
		return entry.conn == conn
	})
	if !ok {
		return common.WrapErr(errors.New("an error occured during pool connection release"))
	}
	atomic.StoreInt32(&entry.acquired, 0)
	p.clientWaiter.Release(1)
	return nil
}

type poolEntry struct {
	acquired int32
	conn     *Connection
}
