package adapter

import (
	"github.com/upwrd/sift/network/ipv4"
	"github.com/upwrd/sift/types"
)

const (
	defaultUpdateChanLen = 100
)

// An Adapter is responsible for two-way sync between a service.
// A service might be a single device (like a WiFi-connected lock), or a
// collection of separate devices (like a hub for Zigbee lights). Each adapter
// handles a single network service, such as a single IP address or Bluetooth
// MAC, and several Adapters may be servicing the same _device_ through
// different services (like a fan that is controllable via both Bluetooth and
// Zigbee protocols).
type Adapter interface {
	EnactIntent(types.ExternalComponentID, types.Intent) error
	UpdateChan() chan interface{}
}

// AdapterFactory is an interface that all factories must satisfy.
// For now, the only requirement is that Factories have names (so we can log
// and debug appropriately).
type AdapterFactory interface {
	Name() string // Factories have names
}

// An IPv4AdapterFactory creates Adapters which control IPv4 services.
type IPv4AdapterFactory interface {
	AdapterFactory
	HandleIPv4(ipv4.ServiceContext) Adapter
	GetIPv4Description() ipv4.ServiceDescription
}
