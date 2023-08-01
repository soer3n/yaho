package hub

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func NewHub() *Hub {
	return &Hub{
		Backends: make(map[string]BackendInterface),
	}
}

func (h *Hub) AddBackend(backend BackendInterface, tickerCtx context.Context, d time.Duration) error {

	b, ok := h.Backends[backend.GetName()]

	if !ok {
		klog.V(0).Infof("set new backend %s. Kubeconfig: %s", backend.GetName(), string(backend.GetConfig()))
		h.Backends[backend.GetName()] = backend
	}

	klog.V(0).Infof("try to start backend %s...", backend.GetName())
	klog.V(0).Infof("%v", b)
	h.Backends[backend.GetName()].Start(tickerCtx, d)
	return nil
}

func (h *Hub) UpdateBackend(backend BackendInterface, secret *v1.Secret) error {

	if _, ok := h.Backends[backend.GetName()]; !ok {
		h.Backends[backend.GetName()] = backend
	} else {
		if err := h.Backends[backend.GetName()].Update(
			backend.GetDefaults(),
			secret,
			backend.GetScheme(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (h *Hub) RemoveBackend(name string) error {
	if _, ok := h.Backends[name]; !ok {
		klog.V(0).Infof("backend for cluster %s already deleted...", name)
		klog.V(0).Infof("current backends in list: %v", h.Backends)
		return nil
	}

	if err := h.Backends[name].Stop(); err != nil {
		klog.V(0).Infof("stopping backend for cluster %s...", name)
		return err
	}

	klog.V(0).Infof("delete backend %s from hub list...", name)
	delete(h.Backends, name)
	return nil
}
