package types

// string constants
const (
	ComponentTypeLightEmitter = "light_emitter"
	IntentTypeSetLightEmitter = "set_light_emitter"
)

// LightEmitter represents a real-world light emitter, like a light bulb or lamp.
type LightEmitter struct {
	BaseComponent
	State LightEmitterState
	Stats *LightEmitterStats
	Specs *LightEmitterSpecs
}

// LightEmitterState represents the state of a real-world light emitter.
type LightEmitterState struct {
	BrightnessInPercent uint8 `db:"brightness_in_percent" json:"brightness_in_percent"`
}

// LightEmitterStats contains statistics about the light emitter.
type LightEmitterStats struct {
	HoursOn int `db:"hours_on"`
}

// LightEmitterSpecs represents the specifications of a real-world light emitter.
type LightEmitterSpecs struct {
	MaxOutputInLumens       int `db:"max_output_in_lumens" json:"max_output_in_lumens"`
	MinOutputInLumens       int `db:"min_output_in_lumens" json:"min_output_in_lumens"`
	ExpectedLifetimeInHours int `db:"expected_lifetime_in_hours" json:"expected_lifetime_in_hours"`
}

// Type returns ComponentTypeLightEmitter. LightEmitter implements types.Component
func (c LightEmitter) Type() string { return ComponentTypeLightEmitter }

// GetTyped returns a typed version of the Component. LightEmitter implements types.Component
func (c LightEmitter) GetTyped() interface{} {
	return struct {
		Type string
		LightEmitter
	}{
		Type:         c.Type(),
		LightEmitter: c,
	}
}

//
// Intents
//

// SetLightEmitterIntent represents an intent to change the light emitter's state
type SetLightEmitterIntent struct {
	BrightnessInPercent uint8 `db:"brightness_in_percent" json:"brightness_in_percent"`
}

// Type returns IntentTypeSetLightEmitter. SetLightEmitterIntent implements types.Intent
func (i SetLightEmitterIntent) Type() string { return IntentTypeSetLightEmitter }

// GetTyped returns a typed version of the Intent. SetLightEmitterIntent implements types.Intent
func (i SetLightEmitterIntent) GetTyped() interface{} {
	return struct {
		Type string
		SetLightEmitterIntent
	}{
		Type: i.Type(),
		SetLightEmitterIntent: i,
	}
}
