package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/AB-Lindex/tsproxy/internal/metrics"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type connection struct {
	proxyservice *proxyservice
	listener     *listener

	inbound  net.Conn
	outbound net.Conn
}

func newConnection(ps *proxyservice, listener *listener, accepted net.Conn) (*connection, error) {

	outbound, err := net.DialTimeout("tcp", fmt.Sprintf("%s.%s:%d", listener.name, listener.namespace, listener.svcPort), 5*time.Second)
	if err != nil {
		accepted.Close()
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

func (conn *connection) Run() {
	metrics.ConnectionOpened()
	go conn.copy(conn.inbound, conn.outbound, metrics.NextWorker())
	go conn.copy(conn.outbound, conn.inbound, metrics.NextWorker())
}

func (conn *connection) copy(from, to net.Conn, workerID int) {
	logger := log.FromContext(context.Background())
	defer logger.Info("Connection closed", "worker", workerID, "from", from.RemoteAddr(), "to", to.RemoteAddr())
	logger.Info("Connection opened", "worker", workerID, "from", from.RemoteAddr(), "remote", to.RemoteAddr())

	// Echo all incoming data.
	_, err := io.Copy(to, from)

	if errors.Is(err, net.ErrClosed) {
		logger.Info("Connection closed", "worker", workerID)
	} else if err != nil {
		logger.Error(err, "Connection error", "worker", workerID)
	}
	// Shut down the connection.
	_ = from.Close()
	_ = to.Close()
}
