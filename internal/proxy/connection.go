package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/AB-Lindex/tsproxy/internal/metrics"
	"github.com/AB-Lindex/tsproxy/internal/options"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type listener struct {
	proxyservice *proxyservice
	key          string
	namespace    string
	name         string
	svcPort      int32
	exposeAsPort int32
	connectTo    string

	listener    net.Listener
	connections map[int]*connection
	mutex       sync.Mutex

	metricsVec []string
}

var netListener = net.ListenConfig{
	KeepAliveConfig: keepalive,
}

func makeConnectionKey(ns, name string, svcPort, tgtPort int32) string {
	return fmt.Sprintf("%s/%s/%d/%d", ns, name, svcPort, tgtPort)
}

func newListener(ps *proxyservice, ctx context.Context, ns, name string, svcPort, tgtPort int32) *listener {
	logger := log.FromContext(ctx)

	key := makeConnectionKey(ns, name, svcPort, tgtPort)

	mvec := metrics.CreateListenerVec(ns, name, svcPort, tgtPort)

	logger.Info("New listener", "key", key, "namespace", ns, "name", name, "port", tgtPort)

	conn := &listener{
		proxyservice: ps,
		key:          key,
		namespace:    ns,
		name:         name,
		svcPort:      svcPort,
		exposeAsPort: tgtPort,
		metricsVec:   mvec,
		connectTo:    fmt.Sprintf("%s.%s:%d", name, ns, svcPort),
	}

	return conn
}

func (conn *listener) Close(ctx context.Context) {
	logger := log.FromContext(ctx)

	logger.Info("Closing connection", "key", conn.key)

	_ = conn.listener.Close()

	delete(conn.proxyservice.listeners, conn.key)
	delete(tsp.ports, conn.exposeAsPort)

	metrics.ListenerClosed(conn.metricsVec)
}

func listen(port int32) (net.Listener, error) {
	address := fmt.Sprintf(":%d", port)
	if options.Flags.Keepalive {
		return netListener.Listen(context.Background(), "tcp", address)
	}
	return net.Listen("tcp", address)
}

func (conn *listener) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)

	logger.Info("Starting listener",
		"key", conn.key,
		"namespace", conn.namespace,
		"name", conn.name,
		"port", conn.exposeAsPort)

	// // connect to service
	// connsvc, err := net.Dial("tcp", fmt.Sprintf("%s.%s:%d", conn.name, conn.namespace, conn.svcPort))
	// if err != nil {
	// 	logger.Error(err, "Failed to connect to service")
	// 	return err
	// }
	// conn.svcConn = connsvc

	// listen on target port
	listener, err := listen(conn.exposeAsPort)

	if err != nil {
		logger.Error(err, "Failed to listen on target port")
		return err
	}
	conn.listener = listener

	tsp.ports[conn.exposeAsPort] = conn

	go conn.Accept(metrics.NextWorker())

	metrics.ListenerOpened(conn.metricsVec)

	return nil
}

func (conn *listener) Accept(workerID int) {
	logger := log.FromContext(context.Background())
	defer logger.Info("Listener closed", "worker", workerID)
	logger.Info("Accepting connections", "key", conn.key, "worker", workerID)

	for {
		accepted, err := conn.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				logger.Error(err, "Listener closed", "key", conn.key)
				return
			}
			logger.Error(err, "Failed to accept connection", "key", conn.key)
			continue
		}

		connect, err := newConnection(conn.proxyservice, conn, accepted)
		if err != nil {
			logger.Error(err, "Failed to create connection", "key", conn.key)
			continue
		}

		a, b := metrics.NextDualWorker()
		conn.AddConnection(a, connect)
		connect.Run(a, b)
	}
}

func (conn *listener) AddConnection(id int, c *connection) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if conn.connections == nil {
		conn.connections = make(map[int]*connection)
	}
	conn.connections[id] = c

	metrics.ConnectionOpened(conn.metricsVec)
}

func (conn *listener) RemoveConnection(id int) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if _, found := conn.connections[id]; !found {
		return
	}

	metrics.ConnectionClosed(conn.metricsVec)

	delete(conn.connections, id)
}
