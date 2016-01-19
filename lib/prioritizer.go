package lib

import (
	"fmt"
	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"
	"sift/types"
	"sort"
	"sync"
)

type controllerType uint8

// The values in this list are used to prioritize services. Lower numbers are higher priority.
const (
	ControllerTypeZigbee     = 1
	ControllerTypeZWave      = 2
	ControllerTypeBluetooth  = 3
	ControllerTypeIPv4       = 4
	ControllerTypeAggregator = 255
)

const (
	outputChanLen = 100
)

// An AdapterDescription describes an adapter for the purpose of prioritization
type AdapterDescription struct {
	Type controllerType
	ID   string
}

// An IPrioritizer considers updates from Adapters and determines whether those
// updates are the highest-priority updates for a particular entity (most
// often, for a particular Device). If so, the update is passed on to any
// listeners.
type IPrioritizer interface {
	OutputChan() chan interface{} // Get the output channel

	// Consider evaluates a device update to see if it's worth passing on to
	// listeners. If there are higher-priority Adapters which are reporting
	// updates for a device, an update from a lower-priority Adapter will be
	// ignored.
	//
	// If the update comes from the highest-priority Adapter for the Device,
	// the update will be passed into the prioritizer's output channel.
	Consider(desc AdapterDescription, update interface{}) error

	// GetHighestPriorityAdapterForDevice returns the highest priority Adapter
	// which is reporting on the state of the Device with id 'id'. An empty string
	// indicates that no such Adapter was found.
	GetHighestPriorityAdapterForDevice(id types.ExternalDeviceID) string
}

// A Prioritizer considers updates from Adapters and determines whether those
// updates are the highest-priority updates for a particular entity (most
// often, for a particular Device). If so, the update is passed on to any
// listeners.
type Prioritizer struct {
	dest chan interface{}
	//sortFn                       func(a1, a2 string) bool
	sortFns                      []lessFunc
	adapterChannelsByToken       map[string]chan interface{}
	rankedAdapterIDsByDeviceID   map[types.ExternalDeviceID][]string
	rankedAdapterDescsByDeviceID map[types.ExternalDeviceID][]AdapterDescription
	rlock                        *sync.Mutex // protects rankedAdapterIDsByDeviceID
	log                          log.Logger
}

func byType(a1, a2 *AdapterDescription) bool { return a1.Type < a2.Type }
func byID(a1, a2 *AdapterDescription) bool   { return a1.ID < a2.ID }

var defaultLessFuncs = []lessFunc{byType}

// NewPrioritizer properly instantiates a Prioritizer
func NewPrioritizer(sortFns []lessFunc) *Prioritizer {
	if sortFns == nil {
		sortFns = defaultLessFuncs
	}
	return &Prioritizer{
		dest:                         make(chan interface{}, outputChanLen),
		sortFns:                      sortFns,
		adapterChannelsByToken:       make(map[string]chan interface{}),
		rankedAdapterIDsByDeviceID:   make(map[types.ExternalDeviceID][]string),
		rankedAdapterDescsByDeviceID: make(map[types.ExternalDeviceID][]AdapterDescription),
		rlock: &sync.Mutex{},
		log:   Log.New("obj", "prioritizer", "id", logext.RandId(8)),
	}
}

// OutputChan returns the output channel of this Prioritizer.
func (p *Prioritizer) OutputChan() chan interface{} {
	return p.dest
}

// Consider evaluates a device update to see if it's worth passing on to
// listeners. If there are higher-priority Adapters which are reporting
// updates for a device, an update from a lower-priority Adapter will be
// ignored.
//
// If the update comes from the highest-priority Adapter for the Device,
// the update will be passed into the prioritizer's output channel.
func (p *Prioritizer) Consider(ad AdapterDescription, update interface{}) error {
	p.log.Debug("considering update", "from_adapter", ad.ID, "adapter_type", ad.Type, "update_type", fmt.Sprintf("%T", update))
	if ad.ID == "" {
		return fmt.Errorf("adapter description must contain non-empty ID")
	}

	if p.dest == nil {
		p.log.Warn("destination for adapterPrioritizer is unset, ignoring update\n", "update", update, "origin_controller", ad)
		return nil // not an error condition(?)
	}

	switch typed := update.(type) {
	default:
		p.log.Warn("got unknown update type", "update_type", fmt.Sprintf("%T", update))
	case DeviceUpdated:
		if p.isHighestPriorityUpdate(ad, typed) {
			p.log.Debug("got update from highest-priority source, putting into desired channel", "source", ad, "update", update, "dest", p.dest, "len(dest)", len(p.dest), "cap(dest)", cap(p.dest))
			p.dest <- typed
		}
	case DeviceDeleted:
		if p.isHighestPriorityDelete(ad, typed) {
			p.log.Debug("ConsiderDesc() putting delete into channel %p (len: %v, cap: %v)", p.dest, len(p.dest), cap(p.dest))
			p.dest <- typed
		}
	}
	return nil
}

func (p *Prioritizer) isHighestPriorityUpdate(ad AdapterDescription, updated DeviceUpdated) bool {
	if p.rankedAdapterDescsByDeviceID == nil {
		panic("adapterPrioritizer was not initialized properly!")
	}

	// Ensure that this adapter token is in the appropriate list
	p.rlock.Lock() // lock
	rankedAdapters, ok := p.rankedAdapterDescsByDeviceID[updated.ID]
	if !ok { // token list for this device doesn't exist yet
		p.log.Debug("received update for device which has not been seen yet", "device_id", updated.ID)
		p.rankedAdapterDescsByDeviceID[updated.ID] = []AdapterDescription{ad}
		rankedAdapters = p.rankedAdapterDescsByDeviceID[updated.ID]
	} else { // existing token list found
		// Ensure that this Adapter's token is in the list
		var found bool
		for _, existingDesc := range rankedAdapters {
			if existingDesc == ad {
				found = true
				break
			}
		}

		if !found {
			p.rankedAdapterDescsByDeviceID[updated.ID] = append(p.rankedAdapterDescsByDeviceID[updated.ID], ad)
			orderedBy(p.sortFns...).sort(p.rankedAdapterDescsByDeviceID[updated.ID])
			rankedAdapters = p.rankedAdapterDescsByDeviceID[updated.ID]
		}
	}
	p.rlock.Unlock()
	p.log.Debug("interpreted an update", "priority_list", p.rankedAdapterDescsByDeviceID, "update_source", updated.ID, "was_source_highest_priority?", rankedAdapters[0] == ad)
	return rankedAdapters[0] == ad // Is this the highest-priority adapter?
}

func (p *Prioritizer) isHighestPriorityDelete(ad AdapterDescription, deleted DeviceDeleted) bool {
	// Get the sorted list of Adapters registered for this device. If the token
	// is first in the list, pass along the update.
	rankedAdapters, ok := p.rankedAdapterDescsByDeviceID[deleted.ID]
	if ok {
		i := 0
		for i < len(rankedAdapters) {
			if rankedAdapters[i] == ad {
				break
			}
			i++
		}

		if i < len(rankedAdapters) && rankedAdapters[i] == ad {
			p.rankedAdapterDescsByDeviceID[deleted.ID] = append(rankedAdapters[:i], rankedAdapters[i+1:]...)
			return i == 0
		}
	}
	return false
}

// GetHighestPriorityAdapterForDevice returns the highest priority Adapter
// which is reporting on the state of the Device with id 'id'. An empty string
// indicates that no such Adapter was found.
func (p *Prioritizer) GetHighestPriorityAdapterForDevice(id types.ExternalDeviceID) string {
	p.rlock.Lock() // lock
	defer p.rlock.Unlock()

	p.log.Debug("finding highest priority adapter for device", "device_key", id, "priority_map", p.rankedAdapterDescsByDeviceID)
	descs, ok := p.rankedAdapterDescsByDeviceID[id]
	if ok && len(descs) > 0 {
		p.log.Debug("prioritizer determined highest priority adapter for update", "device_key", id, "highest_priority", descs[0].ID)
		return descs[0].ID
	}
	p.log.Debug("no highest priority adapter found for update", "device_key", id)
	return ""
}

//
// Sorting functions
//
type lessFunc func(p1, p2 *AdapterDescription) bool

// multiSorter implements the Sort interface, sorting the changes within.
type multiSorter struct {
	descs []AdapterDescription
	less  []lessFunc
}

// Sort sorts the argument slice according to the less functions passed to OrderedBy.
func (ms *multiSorter) sort(descs []AdapterDescription) {
	ms.descs = descs
	sort.Sort(ms)
}

// OrderedBy returns a Sorter that sorts using the less functions, in order.
// Call its Sort method to sort the data.
func orderedBy(less ...lessFunc) *multiSorter {
	return &multiSorter{
		less: less,
	}
}

// Len is part of sort.Interface.
func (ms *multiSorter) Len() int {
	return len(ms.descs)
}

// Swap is part of sort.Interface.
func (ms *multiSorter) Swap(i, j int) {
	ms.descs[i], ms.descs[j] = ms.descs[j], ms.descs[i]
}

// Less is part of sort.Interface. It is implemented by looping along the
// less functions until it finds a comparison that is either Less or
// !Less. Note that it can call the less functions twice per call. We
// could change the functions to return -1, 0, 1 and reduce the
// number of calls for greater efficiency: an exercise for the reader.
func (ms *multiSorter) Less(i, j int) bool {
	p, q := &ms.descs[i], &ms.descs[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		switch {
		case less(p, q):
			// p < q, so we have a decision.
			return true
		case less(q, p):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports.
	return ms.less[k](p, q)
}
