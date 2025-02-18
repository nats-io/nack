package controller

import (
	"sync"
	"testing"
	"time"

	natsservertest "github.com/nats-io/nats-server/v2/test"
	"github.com/stretchr/testify/require"
)

func TestConnPool(t *testing.T) {
	t.Parallel()

	s := natsservertest.RunRandClientPortServer()
	defer s.Shutdown()

	c1 := &NatsConfig{
		ClientName: "Client 1",
		ServerURL:  s.ClientURL(),
	}

	c2 := &NatsConfig{
		ClientName: "Client 1",
		ServerURL:  s.ClientURL(),
	}

	c3 := &NatsConfig{
		ClientName: "Client 2",
		ServerURL:  s.ClientURL(),
	}

	pool := newConnPool(0)

	var conn1, conn2, conn3 *pooledConnection
	var err1, err2, err3 error

	wg := &sync.WaitGroup{}
	wg.Add(3)

	go func() {
		conn1, err1 = pool.Get(c1, true)
		wg.Done()
	}()
	go func() {
		conn2, err2 = pool.Get(c2, true)
		wg.Done()
	}()
	go func() {
		conn3, err3 = pool.Get(c3, true)
		wg.Done()
	}()
	wg.Wait()

	require := require.New(t)

	require.NoError(err1)
	require.NoError(err2)
	require.NoError(err3)

	require.Same(conn1, conn2)
	require.NotSame(conn1, conn3)
	require.NotSame(conn2, conn3)

	conn1.Close()
	conn3.Close()

	time.Sleep(time.Second)

	require.False(conn1.nc.IsClosed())
	require.False(conn2.nc.IsClosed())
	require.True(conn3.nc.IsClosed())

	conn4, err4 := pool.Get(c1, true)
	require.NoError(err4)
	require.Same(conn1, conn4)
	require.Same(conn2, conn4)

	conn2.Close()
	conn4.Close()

	time.Sleep(time.Second)

	require.True(conn1.nc.IsClosed())
	require.True(conn2.nc.IsClosed())
	require.True(conn3.nc.IsClosed())
	require.True(conn4.nc.IsClosed())

	conn5, err5 := pool.Get(c1, true)
	require.NoError(err5)
	require.NotSame(conn1, conn5)

	conn5.Close()

	time.Sleep(time.Second)

	require.True(conn5.nc.IsClosed())
}
