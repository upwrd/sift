package types

// An ExternalDeviceID universally identifies a unique Device. Two separate
// systems (e.g. SmartThings and HomeKit) should use the same DeviceExternalKey
// for identical devices.
type ExternalDeviceID struct {
	Manufacturer string
	ID           string
}

// An ExternalComponentID universally identifies a unique Component.
// Two separate systems (e.g. SmartThings and HomeKit) should use the same
// DeviceExternalKey for identical devices.
type ExternalComponentID struct {
	Device ExternalDeviceID
	Name   string
}

// A DeviceID locally identifies a Device within a particular SIFT service
type DeviceID int64

// A ComponentID locally identifies a Component within a particular SIFT service
type ComponentID struct {
	Name     string
	DeviceID DeviceID `db:"device_id"`
}

// A Device represents a single physical unit which contains zero-or-more
// functional units, called Components.
type Device struct {
	Name string // (Optional) A short, human-readable name for this Device, like "Upward Switch 01A". Should be locally unique
	//	Manufacturer string
	//	ExternalID   string `json:"external_id"` //TODO: should identifiers be moved outside of the struct?
	IsOnline   bool                 `db:"is_online" json:"is_online"`
	Components map[string]Component // All components connected to the Device (indexed by their ID).
}

// DeviceStats describe statistics generated about an individual Device
type DeviceStats struct {
	HoursOnline int64 `db:"hours_online" json:"hours_online"`
}

// A Component represents a single functional element
type Component interface {
	Typeable
	Baseable
}

// An Intent represents a desire for a specific Component to behave in a
// particular way
type Intent interface {
	Typeable
}

// A Typeable element has a type, and can return a 'Typed' version of itself,
// which is an identical struct with an added field `Type string`. This is used
// for marshalling and unmarshalling polymorphic collections.
type Typeable interface {
	Type() string          // return a string indicating the type
	GetTyped() interface{} // return a typed version of this entity
}

// BaseComponent describes the shared attributes of each Componenent
type BaseComponent struct {
	//DeviceID string `json:"device_id"`
	Make  string
	Model string
}

// A Baseable struct can produce a BaseComponent.
type Baseable interface {
	GetBaseComponent() BaseComponent
}

// GetBaseComponent returns the Component's BaseComponent
func (b BaseComponent) GetBaseComponent() BaseComponent { return b }
