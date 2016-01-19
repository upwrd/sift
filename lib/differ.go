package lib

import (
	"fmt"
	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"
	"reflect"
	"github.com/upwrd/sift/types"
	"sync"
)

// DeviceUpdated describes a single Device which has been updated
type DeviceUpdated struct {
	ID       types.ExternalDeviceID
	NewState types.Device
}

// DeviceDeleted describes a single Device which has been deleted
type DeviceDeleted struct {
	ID types.ExternalDeviceID
}

// A SetOutputBasedDeviceDiffer is a DeviceDiffer which can be set to output
// results to a given channel
type SetOutputBasedDeviceDiffer interface {
	DeviceDiffer
	SetOutput(dest chan interface{}) error
}

// A ChannelBasedDeviceDiffer is a DeviceDiffer which returns a results channel
type ChannelBasedDeviceDiffer interface {
	DeviceDiffer
	OutputChan() chan interface{}
}

// A DeviceDiffer maintains a state of Device sets which have been considered.
// Each time a new Device set is considered, it determines whether or not that
// set indicates that one or more Devices have been added, updated, or removed
// from the caller's service. If so, those updates are sent to a destination
// channel to be consumed.
type DeviceDiffer interface {
	Consider(map[types.ExternalDeviceID]types.Device)
	GetLatest(types.ExternalDeviceID) (types.Device, error)
}

// An AllAtOnceDiffer is a DeviceDiffer that expects a caller to Consider all
// of it's Devices every time it is called. It compares each full device set
// against the most recent one - if a new Device set contains a Device that was
// not seen, this is considered to be a new Device. Similarly, if a new Device
// set does not contain a Device that was previously seen, that Device is
// considered to have been deleted.
type AllAtOnceDiffer struct {
	lastKnownDevices map[types.ExternalDeviceID]types.Device
	lock             *sync.Mutex // protects lastKnownDevices
	dest             chan interface{}
	log              log.Logger
}

// NewAllAtOnceDiffer properly instantiates an AllAtOnceDiffer
func NewAllAtOnceDiffer() *AllAtOnceDiffer {
	return &AllAtOnceDiffer{
		lastKnownDevices: make(map[types.ExternalDeviceID]types.Device),
		lock:             &sync.Mutex{},
		log:              Log.New("obj", "differ", "id", logext.RandId(8)),
	}
}

// SetOutput sets the output destination channel for the AllAtOnceDiffer
func (d *AllAtOnceDiffer) SetOutput(dest chan interface{}) error {
	if d.dest != nil {
		return fmt.Errorf("output already set (channel %v)", d.dest)
	}
	if dest == nil {
		return fmt.Errorf("destination channel cannot be nil")
	}
	d.dest = dest
	return nil
}

// OutputChan gets the output destination channel for the AllAtOnceDiffer
func (d *AllAtOnceDiffer) OutputChan() chan interface{} {
	return d.dest
}

// Consider considers a new Device Set and determines if that set indicates
// that one or more Devices have been added, updated, or removed from the
// caller's service. If so, those updates are sent to a destination channel to
// be consumed.
func (d *AllAtOnceDiffer) Consider(devices map[types.ExternalDeviceID]types.Device) {
	if d == nil {
		fmt.Printf("ERROR: called Consider with nil allAtOnceDiffer")
		return
	}
	if d.dest == nil {
		d.log.Error("destination was not; ignoring call to Consider()")
		return // ignore if the destination channel was set improperly
	}
	d.log.Debug("differ considering devices", "output_chan", d.dest, "devices_to_consider", devices)

	d.lock.Lock()
	defer d.lock.Unlock()

	var deviceUpdates []DeviceUpdated
	var deviceDeletes []DeviceDeleted

	for id, device := range devices {
		if lastKnown, ok := d.lastKnownDevices[id]; !ok || !reflect.DeepEqual(lastKnown, device) {
			update := DeviceUpdated{
				ID:       id,
				NewState: device,
			}
			deviceUpdates = append(deviceUpdates, update)
		}
		delete(d.lastKnownDevices, id) // delete marks that the device has been considered
	}

	for id := range d.lastKnownDevices {
		delete := DeviceDeleted{
			ID: id,
		}
		deviceDeletes = append(deviceDeletes, delete)
	}
	d.lastKnownDevices = devices // The considered list is now the last-known state

	for _, update := range deviceUpdates {
		d.log.Debug("putting update into output channel", "output_chan", d.dest, "update", update)
		d.dest <- update
	}
	for _, delete := range deviceDeletes {
		d.log.Debug("putting delete into output channel", "output_chan", d.dest, "delete", delete)
		d.dest <- delete
	}
	if len(devices) > (len(deviceUpdates) + len(deviceDeletes)) {
		d.log.Debug("some devices were not considered different", "num_undifferent_devices", len(devices)-(len(deviceUpdates)+len(deviceDeletes)))
	}
}

// GetLatest returns the latest-considered Device which matches the given
// types.ExternalDeviceID
func (d *AllAtOnceDiffer) GetLatest(id types.ExternalDeviceID) (types.Device, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if device, ok := d.lastKnownDevices[id]; ok {
		return device, nil
	}
	return types.Device{}, fmt.Errorf("cannot find device with key %v", id)
}

//
// Component Set Differ
//

type componentDiffResults struct {
	Upserted map[string]types.Component
	Deleted  []string
}

func copyMap(original map[string]types.Component) map[string]types.Component {
	newMap := make(map[string]types.Component)
	for key, val := range original {
		newMap[key] = val
	}
	return newMap
}

func diffComponents(old, new map[string]types.Component) (upserted, deleted map[string]types.Component) {
	oldCopy, newCopy := copyMap(old), copyMap(new)

	upserted = make(map[string]types.Component)
	for id, newComponent := range newCopy {
		oldComponent, ok := oldCopy[id]
		if !ok {
			// No matching old component found - this new component is an update
			upserted[id] = newComponent
		}
		if !reflect.DeepEqual(newComponent, oldComponent) {
			upserted[id] = newComponent
		}
		delete(oldCopy, id)
	}

	deleted = make(map[string]types.Component)
	for id, comp := range oldCopy {
		deleted[id] = comp
	}

	return
}

// DiffDevice determines the differences between one 'old' Device and a 'new'
// Device, in the context that the 'new' Device is replacing the 'old' Device.
// It lists the components that would be upserted or deleted, and determines if
// the Device itself has changed.
func DiffDevice(old, new types.Device) (upserted, deleted map[string]types.Component, deviceChanged bool) {
	upserted, deleted = diffComponents(old.Components, new.Components)
	deviceChanged = old.Name != new.Name
	return
}
