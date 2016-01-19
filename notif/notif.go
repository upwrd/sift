package notif

import (
	"fmt"
	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"
	"sift/auth"
	"sift/logging"
	"sift/types"
	"strings"
	"sync"
)

// Log is used to log messages for the sift package. Logs are disabled by
// default; use sift/logging.SetLevel() to set log levels for all packages, or
// Log.SetHandler() to set a custom handler for this package (see:
// https://godoc.org/gopkg.in/inconshreveable/log15.v2)
var Log = logging.Log.New("pkg", "notif")

// An ActionsMask represents actions which may trigger a notification.
type ActionsMask uint8

// Possible ActionTypes (powers of two for bitwise arithmetic)
const (
	All    = ActionsMask(0)
	Create = ActionsMask(1)
	Update = ActionsMask(2)
	Delete = ActionsMask(4)
	Moved  = ActionsMask(8)
)

const chanCap = 100 // The capacity of returned notification channels

// A Provider provides methods for listeners to listen for notifications
type Provider interface {
	Listen(auth.Token, ...interface{}) <-chan interface{}
	//Unlisten(<-chan interface{})
}

// A Receiver provides methods for posting notifications to listeners
type Receiver interface {
	ComponentNotifier
	DeviceNotifier
}

// A ProviderReceiver provides methods for listeners to listen for notifications,
// as well as methods to post notifications to listeners.
type ProviderReceiver interface {
	Provider
	Receiver
}

// A Notifier implements ProviderReceiver. It provides methods for listeners to
// listen for notifications, as well as methods to post notifications to
// listeners.
type Notifier struct {
	authorizor auth.Authorizor

	lock         sync.RWMutex
	channelLocks map[chan interface{}]sync.Mutex

	// filtersByChannels records the filters used to set up each channel so we can undo them
	filtersByChanel map[chan interface{}][]interface{}
	// authTokenByChannel records the auth token used to create each channel
	authTokenByChannel map[chan interface{}]auth.Token

	// Indices for unrestricted
	allNotificationListeners map[chan interface{}]ActionsMask

	// Indices for Drivers
	driverListenersFilteredByType map[string]map[chan interface{}]ActionsMask
	driverListenersFilteredByID   map[string]map[chan interface{}]ActionsMask
	unfilteredDriverListeners     map[chan interface{}]ActionsMask

	// Indices for Devices
	deviceListenersFilteredByType map[string]map[chan interface{}]ActionsMask
	deviceListenersFilteredByID   map[types.DeviceID]map[chan interface{}]ActionsMask
	unfilteredDeviceListeners     map[chan interface{}]ActionsMask

	// Indices for Components
	componentListenersFilteredByType map[string]map[chan interface{}]ActionsMask
	componentListenersFilteredByID   map[types.ComponentID]map[chan interface{}]ActionsMask
	unfilteredComponentListeners     map[chan interface{}]ActionsMask

	log log.Logger
}

// New creates a new notifier.
func New(authorizor auth.Authorizor) *Notifier {
	return &Notifier{
		authorizor: authorizor,

		lock:         sync.RWMutex{},
		channelLocks: make(map[chan interface{}]sync.Mutex),

		filtersByChanel:    make(map[chan interface{}][]interface{}),
		authTokenByChannel: make(map[chan interface{}]auth.Token),

		allNotificationListeners: make(map[chan interface{}]ActionsMask),

		// Indices for Drivers
		driverListenersFilteredByType: make(map[string]map[chan interface{}]ActionsMask),
		driverListenersFilteredByID:   make(map[string]map[chan interface{}]ActionsMask),
		unfilteredDriverListeners:     make(map[chan interface{}]ActionsMask),

		// Indices for Devices
		deviceListenersFilteredByType: make(map[string]map[chan interface{}]ActionsMask),
		deviceListenersFilteredByID:   make(map[types.DeviceID]map[chan interface{}]ActionsMask),
		unfilteredDeviceListeners:     make(map[chan interface{}]ActionsMask),

		componentListenersFilteredByType: make(map[string]map[chan interface{}]ActionsMask),
		componentListenersFilteredByID:   make(map[types.ComponentID]map[chan interface{}]ActionsMask),
		unfilteredComponentListeners:     make(map[chan interface{}]ActionsMask),

		log: Log.New("obj", "notifier", "id", logext.RandId(8)),
	}
}

// Listen returns a channel which will be populated with notifications from
// the notifier. If one or more filters are provided, only notifications
// matching those filters will populate the channel. If no filters are
// provided, all notifications will populate the channel.
func (n *Notifier) Listen(token auth.Token, filters ...interface{}) <-chan interface{} {
	if n == nil {
		return nil
	}

	nChan := make(chan interface{}, chanCap) // new channel for notifications
	n.lock.Lock()
	defer n.lock.Unlock()
	n.authTokenByChannel[nChan] = token  // save the token for later authentication
	n.filtersByChanel[nChan] = filters   // save filters list so we can undo on unsubscribe
	n.channelLocks[nChan] = sync.Mutex{} // create a lock for this channel

	// If no filters were provided, this should listen to -everything-
	if len(filters) == 0 {
		n.allNotificationListeners[nChan] = All
	}

	// Record each filter so we can find it during notification posts
	for _, filter := range filters {
		// If the filter is a string, try parsing it into a filter struct
		if asStr, ok := filter.(string); ok {
			filter = parseStr(asStr)
		}

		switch typed := filter.(type) {
		case ComponentFilter:
			n.addComponentListener(nChan, typed)
		default:
			n.log.Warn("unhandled filter type", "filter_type", fmt.Sprintf("%T", filter))
		}
	}
	return nChan
}

// doPost posts a notification to a channel. If it is full, doPost will drop the
// notification and print a log warning.
func (n *Notifier) doPost(nchan chan interface{}, val interface{}) {
	if n == nil {
		return
	}
	if lock, ok := n.channelLocks[nchan]; ok {
		lock.Lock()
		if len(nchan) < cap(nchan) {
			n.log.Debug("posting notification to channel", "chan", nchan, "value", val)
			nchan <- val
		} else {
			log.Warn("dropping notification to channel because it is full", "chan", nchan)
		}
		lock.Unlock()
	} else {
		log.Warn("dropping notification to channel because a matching channel lock was not found", "chan", nchan)
	}
}

// parseStr parses a string into a notification registration type.
// If no matches exist, returns nil.
func parseStr(str string) interface{} {
	switch str {
	case "components":
		return ComponentFilter{}
	default:
		return nil
	}
}

// String returns a human-readable string representation of the ActionsMask
func (a ActionsMask) String() string {
	actionStr := ""
	if a&Create != 0 {
		actionStr += " create "
	}
	if a&Update != 0 {
		actionStr += " update "
	}
	if a&Delete != 0 {
		actionStr += " delete "
	}
	if a&Moved != 0 {
		actionStr += " moved "
	}
	return strings.TrimSpace(actionStr)
}
