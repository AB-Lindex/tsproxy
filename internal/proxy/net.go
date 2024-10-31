package proxy

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/AB-Lindex/tsproxy/internal/options"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type connection struct {
	proxyservice *proxyservice
	listener     *listener

	inbound  net.Conn
	outbound net.Conn
}

var keepalive = net.KeepAliveConfig{
	Enable:   true,
	Idle:     10 * time.Second,
	Interval: 10 * time.Second,
	Count:    5,
}

var dialer = &net.Dialer{
	Timeout:         5 * time.Second,
	KeepAliveConfig: keepalive,
}

func dial(address string) (net.Conn, error) {
	if options.Flags.Keepalive {
		return dialer.Dial("tcp", address)
	}
	return net.DialTimeout("tcp", address, 5*time.Second)
}

func newConnection(ps *proxyservice, listener *listener, accepted net.Conn) (*connection, error) {
	outbound, err := dial(listener.connectTo)
	if err != nil {
		_ = accepted.Close()
		return nil, err
	}

	conn := &connection{
		proxyservice: ps,
		listener:     listener,
		inbound:      accepted,
		outbound:     outbound,
	}

	return conn, nil
}

func (conn *connection) Run(a, b int) {
	logger := log.FromContext(context.Background())
	logger.Info("Connection opened",
		"key", conn.listener.key,
		"worker", a,
		"from", conn.inbound.RemoteAddr().String())
	// "remote", conn.outbound.RemoteAddr().String())
	go conn.copy(conn.inbound, conn.outbound, a, a)
	go conn.copy(conn.outbound, conn.inbound, b, a)
}

func (conn *connection) copy(from, to net.Conn, workerID, primaryID int) {
	logger := log.FromContext(context.Background())
	if workerID == primaryID {
		defer logger.Info("Connection closed",
			"key", conn.listener.key,
			"worker", workerID,
			"from", from.RemoteAddr().String())
	}
	defer conn.listener.RemoveConnection(primaryID)

	// Echo all incoming data.
	_, err := io.Copy(to, from)

	if errors.Is(err, net.ErrClosed) {
		logger.Info("Connection closing", "key", conn.listener.key, "worker", workerID)
	} else if err != nil {
		logger.Error(err, "Connection error", "key", conn.listener.key, "worker", workerID)
	}
	// Shut down the connection.
	_ = from.Close()
	_ = to.Close()
}
