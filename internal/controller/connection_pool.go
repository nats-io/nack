package controller

import (
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type pooledConnection struct {
	nc       *nats.Conn
	pool     *connectionPool
	hash     string
	refCount int
}

func (pc *pooledConnection) Close() {
	if pc.pool != nil {
		pc.pool.release(pc.hash)
	} else if pc.nc != nil {
		pc.nc.Close() // Close directly if not pool-managed
	}
}

type connectionPool struct {
	connections map[string]*pooledConnection
	gracePeriod time.Duration
	mu          sync.Mutex
}

func newConnPool(gracePeriod time.Duration) *connectionPool {
	return &connectionPool{
		connections: make(map[string]*pooledConnection),
		gracePeriod: gracePeriod,
	}
}

func (p *connectionPool) Get(c *NatsConfig, pedantic bool) (*pooledConnection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	hash, err := c.Hash()
	if err != nil {
		// If hash fails, create a new non-pooled connection
		nc, err := createNatsConn(c)
		if err != nil {
			return nil, err
		}
		return &pooledConnection{nc: nc}, nil
	}

	if pc, ok := p.connections[hash]; ok && !pc.nc.IsClosed() {
		pc.refCount++
		return pc, nil
	}

	nc, err := createNatsConn(c)
	if err != nil {
		return nil, err
	}

	pc := &pooledConnection{
		nc:       nc,
		pool:     p,
		hash:     hash,
		refCount: 1,
	}
	p.connections[hash] = pc

	return pc, nil
}

func (p *connectionPool) release(hash string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	pc, ok := p.connections[hash]
	if !ok {
		return
	}

	pc.refCount--
	if pc.refCount < 1 {
		go func() {
			if p.gracePeriod > 0 {
				time.Sleep(p.gracePeriod)
			}

			p.mu.Lock()
			defer p.mu.Unlock()

			if pc, ok := p.connections[hash]; ok && pc.refCount < 1 {
				pc.nc.Close()
				delete(p.connections, hash)
			}
		}()
	}
}
