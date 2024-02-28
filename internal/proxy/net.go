package proxy

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type connection struct {
	proxyservice *proxyservice
	listener     *listener

	inbound  net.Conn
	outbound net.Conn
}

func newConnection(ps *proxyservice, listener *listener, accepted net.Conn) (*connection, error) {

	outbound, err := net.DialTimeout("tcp", listener.connectTo, 5*time.Second)
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
			"from", from.RemoteAddr().String(),
			"to", to.RemoteAddr().String())
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
