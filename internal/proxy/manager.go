package proxy

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"

	proxyv1alpha1 "github.com/AB-Lindex/tsproxy/api/v1alpha1"
	"github.com/AB-Lindex/tsproxy/internal/options"
)

const (
	PORTNO_MIN = 1
	PORTNO_MAX = 65535
)

type manager struct {
	active map[string]*proxyservice
	ports  map[int32]*listener
}

type proxyservice struct {
	key       types.NamespacedName
	obj       *proxyv1alpha1.TSProxy
	listeners map[string]*listener
}

var tsp = &manager{
	active: make(map[string]*proxyservice),
	ports:  make(map[int32]*listener),
}

func Reload(ctx context.Context, key types.NamespacedName, obj *proxyv1alpha1.TSProxy) {
	if options.Flags.Debug {
		defer tsp.Dump(ctx)
	}

	// y := yaml.NewEncoder(os.Stdout)
	// y.Encode(obj)
	// y.Close()

	logger := log.FromContext(ctx)

	logger.Info("Reload", "namespace", key.Namespace, "name", key.Name)

	if obj == nil {
		tsp.Close(ctx, key.String())
		return
	}

	tsp.AddOrUpdate(ctx, key, obj)
}

func (m *manager) IsPortAvailable(port int32) bool {
	_, found := m.ports[port]
	return !found
}

func (m *manager) Validate(ctx context.Context, objKey string, obj *proxyv1alpha1.TSProxy) error {

	for _, svc := range obj.Spec.Services {
		if svc.ExposeAs < PORTNO_MIN || svc.ExposeAs > PORTNO_MAX {
			return fmt.Errorf("ExposeAs %d is out of range", svc.ExposeAs)
		}

		if activeListener, found := m.ports[svc.ExposeAs]; found {
			if activeListener == nil {
				panic("activeListener is nil")
			}
			if activeListener.proxyservice == nil {
				panic("activeListener.proxyservice is nil")
			}
			if activeListener.proxyservice.key.String() != objKey {
				return fmt.Errorf("ExposeAs %d is already in use by %s", svc.ExposeAs, activeListener.proxyservice.key)
			}
		}
	}

	return nil
}

// manager functions
func (m *manager) Close(ctx context.Context, key string) {
	logger := log.FromContext(ctx)

	if _, ok := m.active[key]; !ok {
		logger.Info("TSProxy not found in active list")
		return
	}

	logger.Info("Closing TSProxy")
	svc := m.active[key]
	if svc.Close(ctx) {
		delete(m.active, key)
	}
}

func (m *manager) AddOrUpdate(ctx context.Context, key types.NamespacedName, obj *proxyv1alpha1.TSProxy) {
	logger := log.FromContext(ctx)

	if err := m.Validate(ctx, key.String(), obj); err != nil {
		logger.Error(err, "TSProxy validation failed", "namespace", key.Namespace, "name", key.Name)
		return
	}

	if ps, ok := m.active[key.String()]; ok {
		m.update(ctx, key, obj, ps)
		return
	}

	m.add(ctx, key, obj)
}

func (m *manager) add(ctx context.Context, key types.NamespacedName, obj *proxyv1alpha1.TSProxy) {
	logger := log.FromContext(ctx)

	logger.Info("Adding new proxy", "namespace", key.Namespace, "name", key.Name)
	svc := &proxyservice{
		key:       key,
		obj:       obj,
		listeners: make(map[string]*listener),
	}
	m.active[key.String()] = svc
	svc.Start(ctx)
}

func (m *manager) update(ctx context.Context, key types.NamespacedName, obj *proxyv1alpha1.TSProxy, ps *proxyservice) {
	logger := log.FromContext(ctx)

	logger.Info("Updating existing proxy", "namespace", key.Namespace, "name", key.Name)

	var inUse = make(map[string]bool)
	for key := range ps.listeners {
		inUse[key] = true
	}

	for _, svc := range obj.Spec.Services {
		key := makeConnectionKey(key.Namespace, svc.Name, svc.ServicePort, svc.ExposeAs)
		if _, found := inUse[key]; found {
			logger.Info("Service already running - dont touch", "key", key)
			delete(inUse, key)
			continue
		}
	}
	for svc := range inUse {
		logger.Info("Service no longer in use - closing", "key", svc)
		ps.listeners[svc].Close(ctx)
		delete(ps.listeners, svc)
	}

	var toStart []proxyv1alpha1.TSProxyService
	for _, svc := range obj.Spec.Services {
		key := makeConnectionKey(key.Namespace, svc.Name, svc.ServicePort, svc.ExposeAs)
		if _, found := ps.listeners[key]; !found {
			toStart = append(toStart, svc)
		}
	}

	ps.beginToListen(ctx, toStart)
}

// service functions
func (ps *proxyservice) Close(ctx context.Context) bool {
	logger := log.FromContext(ctx)

	if ps.obj == nil {
		logger.Error(nil, "TSProxy object is nil - unable to close")
		return false
	}

	for _, conn := range ps.listeners {
		conn.Close(ctx)
	}

	return true
}

func (ps *proxyservice) Start(ctx context.Context) {
	logger := log.FromContext(ctx)

	if ps.obj == nil {
		logger.Error(nil, "TSProxy object is nil - unable to start")
		return
	}

	logger.Info("Starting", "namespace", ps.key.Namespace, "name", ps.key.Name, "services", len(ps.obj.Spec.Services))
	ps.beginToListen(ctx, ps.obj.Spec.Services)
}

func (ps *proxyservice) beginToListen(ctx context.Context, services []proxyv1alpha1.TSProxyService) {
	logger := log.FromContext(ctx)

	var newListeners = make([]*listener, 0, len(services))

	for _, svc := range services {
		logger.Info("Starting TSProxy service", "service", svc.Name)
		conn := newListener(ps, ctx, ps.key.Namespace, svc.Name, svc.ServicePort, svc.ExposeAs)
		newListeners = append(newListeners, conn)
	}

	for _, conn := range newListeners {
		if err := conn.Start(ctx); err == nil {
			ps.listeners[conn.key] = conn
		}
	}
}

func (m *manager) Dump(ctx context.Context) {
	logger := log.FromContext(ctx)

	logger.Info("Dumping active connections")
	for _, svc := range m.active {
		logger.Info("Dump: TSProxy", "namespace", svc.key.Namespace, "name", svc.key.Name)

		for _, conn := range svc.listeners {
			logger.Info("Dump: TSProxy service",
				"namespace", conn.namespace,
				"name", conn.name,
				"port", conn.exposeAsPort,
				"ps", conn.proxyservice)
		}
	}
	for port, conn := range m.ports {
		logger.Info("Dump: TSProxy port", "port", port, "namespace", conn.namespace, "name", conn.name)
	}
}
