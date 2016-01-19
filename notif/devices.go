package notif

import "github.com/upwrd/sift/types"

// A DeviceNotifier can notify listeners of changes to Devices
type DeviceNotifier interface {
	PostDevice(id types.DeviceID, dev types.Device, atype ActionsMask)
}

// A DeviceFilter is used by a listener to select noficiations from specific
// Devices. Nil values are interpreted as "don't care".
type DeviceFilter struct {
	ID      types.DeviceID
	Actions ActionsMask
}

// A DeviceNotification describes a change to a single Device
type DeviceNotification struct {
	ID     types.DeviceID
	Device types.Device
	Action ActionsMask
}

func (n *Notifier) addDeviceListener(nchan chan interface{}, filter DeviceFilter) {
	if n == nil {
		return
	}

	// Add the listener to the most appropriate list, based on values in the filter
	switch {
	case filter.ID != 0: // User specified an ID (type, if provided, will be ignored)
		if _, ok := n.deviceListenersFilteredByID[filter.ID]; ok {
			// Listeners already exist for this device; add this new channel to the list
			n.deviceListenersFilteredByID[filter.ID][nchan] = filter.Actions
		} else {
			// This is the first listener for this device; create a new map.
			n.deviceListenersFilteredByID[filter.ID] = map[chan interface{}]ActionsMask{nchan: filter.Actions}
		}
	default: // User did not specify a type or ID, so they will listen to all devices
		n.unfilteredDeviceListeners[nchan] = filter.Actions
	}
}

// PostDevice will notify all listeners of a change to the provided Device. The
// specific type of change should by provided in the ActionsMask.
func (n *Notifier) PostDevice(id types.DeviceID, dev types.Device, atype ActionsMask) {
	nchans := make(map[chan interface{}]struct{}) // A list of channels to notify

	// Get all of the notification channels that match this device & action
	n.lock.RLock()
	defer n.lock.RUnlock()

	// Get notification channels listening for devices with matching IDs
	if filterList, ok := n.deviceListenersFilteredByID[id]; ok {
		for nchan, atypes := range filterList {
			// atypes == 0 means the filter is listening to all actions
			// atypes & atype should be nonzero if atypes contains the bit representing atype
			if atypes == 0 || atypes&atype != 0 {
				nchans[nchan] = struct{}{}
			}
		}
	}

	// Get notification channels listening for all devices
	for nchan, atypes := range n.unfilteredDeviceListeners {
		// atypes == 0 means the filter is listening to all actions
		// atypes & atype should be nonzero if atypes contains the bit representing atype
		if atypes == 0 || atypes&atype != 0 {
			nchans[nchan] = struct{}{}
		}
	}

	// Get notification channels listening for any-and-all notifications
	for nchan, atypes := range n.allNotificationListeners {
		// atypes == 0 means the filter is listening to all actions
		// atypes & atype should be nonzero if atypes contains the bit representing atype
		if atypes == 0 || atypes&atype != 0 {
			nchans[nchan] = struct{}{}
		}
	}

	n.log.Debug("channels during PostDevice", "deviceListenersFilteredByID", n.deviceListenersFilteredByID, "deviceListenersFilteredByType", n.deviceListenersFilteredByType, "unfilteredDeviceListeners", n.unfilteredDeviceListeners, "allNotificationListeners", n.allNotificationListeners)
	n.log.Debug("matching (but not yet authorized) channels", "nchans", nchans)

	// Post to authorized channels
	cnotif := DeviceNotification{
		ID:     id,
		Device: dev,
		Action: atype,
	}
	for nchan := range nchans {
		if token, ok := n.authTokenByChannel[nchan]; ok {
			if n.authorizor.Authorize(token, "devices:-unimplemented-data-type:") {
				n.doPost(nchan, cnotif)
			}
		}
	}
}
