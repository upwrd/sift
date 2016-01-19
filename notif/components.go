package notif

import "github.com/upwrd/sift/types"

// A ComponentNotifier can notify listeners of changes to Components
type ComponentNotifier interface {
	PostComponent(id types.ComponentID, comp types.Component, amask ActionsMask)
}

// A ComponentFilter is used by a listener to select noficiations from specific
// Components. Nil values are interpreted as "don't care".
type ComponentFilter struct {
	ID      types.ComponentID
	Type    string
	Actions ActionsMask
}

// A ComponentNotification describes a change to a single Component
type ComponentNotification struct {
	ID        types.ComponentID
	Component types.Component
	Action    ActionsMask
}

func (n *Notifier) addComponentListener(nchan chan interface{}, filter ComponentFilter) {
	if n == nil {
		return
	}

	// Add the listener to the most appropriate list, based on values in the filter
	switch {
	case filter.ID != types.ComponentID{}: // User specified an ID (type, if provided, will be ignored)
		if _, ok := n.componentListenersFilteredByID[filter.ID]; ok {
			// Listeners already exist for this component; add this new channel to the list
			n.componentListenersFilteredByID[filter.ID][nchan] = filter.Actions
		} else {
			// This is the first listener for this component; create a new map.
			n.componentListenersFilteredByID[filter.ID] = map[chan interface{}]ActionsMask{nchan: filter.Actions}
		}
	case filter.Type != "": // User specified a type
		if _, ok := n.componentListenersFilteredByType[filter.Type]; ok {
			// Filters already exist for this component type; add this new channel to the list
			n.componentListenersFilteredByType[filter.Type][nchan] = filter.Actions
		} else {
			// This is the first listener for this component type; create a new map.
			n.componentListenersFilteredByType[filter.Type] = map[chan interface{}]ActionsMask{nchan: filter.Actions}
		}
	default: // User did not specify a type or ID, so they will listen to all components
		n.unfilteredComponentListeners[nchan] = filter.Actions
	}
}

// PostComponent will notify all listeners of a change to the provided
// Component. The specific type of change should by provided in the
// ActionsMask.
func (n *Notifier) PostComponent(id types.ComponentID, comp types.Component, amask ActionsMask) {
	nchans := make(map[chan interface{}]struct{}) // A list of channels to notify

	// Get all of the notification channels that match this component & action
	n.lock.RLock()
	defer n.lock.RUnlock()

	// Get notification channels listening for components with matching IDs
	if filterList, ok := n.componentListenersFilteredByID[id]; ok {
		for nchan, atypes := range filterList {
			// atypes == 0 means the filter is listening to all actions
			// atypes & atype should be nonzero if atypes contains the bit representing atype
			if atypes == 0 || atypes&amask != 0 {
				nchans[nchan] = struct{}{}
			}
		}
	}

	// Get notification channels listening for components with matching types
	if filterList, ok := n.componentListenersFilteredByType[comp.Type()]; ok {
		for nchan, atypes := range filterList {
			// atypes == 0 means the filter is listening to all actions
			// atypes & atype should be nonzero if atypes contains the bit representing atype
			if atypes == 0 || atypes&amask != 0 {
				nchans[nchan] = struct{}{}
			}
		}
	}

	// Get notification channels listening for all components
	for nchan, atypes := range n.unfilteredComponentListeners {
		// atypes == 0 means the filter is listening to all actions
		// atypes & atype should be nonzero if atypes contains the bit representing atype
		if atypes == 0 || atypes&amask != 0 {
			nchans[nchan] = struct{}{}
		}
	}

	// Get notification channels listening for any-and-all notifications
	for nchan, atypes := range n.allNotificationListeners {
		// atypes == 0 means the filter is listening to all actions
		// atypes & atype should be nonzero if atypes contains the bit representing atype
		if atypes == 0 || atypes&amask != 0 {
			nchans[nchan] = struct{}{}
		}
	}

	n.log.Debug("channels during PostComponent", "componentListenersFilteredByID", n.componentListenersFilteredByID, "componentListenersFilteredByType", n.componentListenersFilteredByType, "unfilteredComponentListeners", n.unfilteredComponentListeners, "allNotificationListeners", n.allNotificationListeners)
	n.log.Debug("matching (but not yet authorized) channels", "nchans", nchans)

	// Post to authorized channels
	cnotif := ComponentNotification{
		ID:        id,
		Component: comp,
		Action:    amask,
	}
	for nchan := range nchans {
		if token, ok := n.authTokenByChannel[nchan]; ok {
			if n.authorizor.Authorize(token, "components:-unimplemented-data-type:") {
				n.doPost(nchan, cnotif)
			}
		}
	}
}
