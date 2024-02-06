package metrics

import (
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type metricsExporter struct {
	initDone bool

	workerValue int
	workerMutex sync.Mutex
	workers     prometheus.Counter

	connectionsTotal prometheus.Counter

	listeners *prometheus.GaugeVec
}

var me = &metricsExporter{}

func initMetrics() {
	if me.initDone {
		return
	}

	me.workers = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tsproxy_worker_total",
		Help: "Total number of workers",
	})
	_ = metrics.Registry.Register(me.workers)

	me.connectionsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "tsproxy_connection_total",
		Help: "Total number of connections",
	})
	_ = metrics.Registry.Register(me.connectionsTotal)

	me.listeners = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tsproxy_listener_active",
		Help: "Active listeners",
	}, []string{"namespace", "name", "serviceport", "targetport"})
	_ = metrics.Registry.Register(me.listeners)

	me.initDone = true
}

func NextWorker() int {
	initMetrics()

	me.workerMutex.Lock()
	defer me.workerMutex.Unlock()

	me.workerValue++
	me.workers.Inc()
	return me.workerValue
}

func ConnectionOpened() {
	initMetrics()
	me.connectionsTotal.Inc()
}

func CreateListenerVec(ns, name string, svcPort, tgtPort int32) []string {
	initMetrics()
	return []string{ns, name, strconv.Itoa(int(svcPort)), strconv.Itoa(int(tgtPort))}
}

func ListenerOpened(vec []string) {
	initMetrics()
	me.listeners.WithLabelValues(vec...).Set(1)
}

func ListenerClosed(vec []string) {
	initMetrics()
	me.listeners.DeleteLabelValues(vec...)
}
