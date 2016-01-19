package types

// string constants
const (
	ComponentTypeSpeaker = "speaker"
	IntentTypeSetSpeaker = "set_speaker"
)

// Speaker represents a real-world speaker.
type Speaker struct {
	BaseComponent
	State SpeakerState
	Specs *SpeakerSpecs
}

// SpeakerState represents the state of a real-world speaker.
type SpeakerState struct {
	IsOnline        bool
	OutputInPercent uint8
}

// SpeakerSpecs represents the specifications of a real-world speaker.
type SpeakerSpecs struct {
	MaxOutputInDecibels     int
	MinOutputInDecibels     int
	ExpectedLifetimeInHours int
}

// Type returns ComponentTypeSpeaker. Speaker implements types.Component
func (c Speaker) Type() string { return ComponentTypeSpeaker }

// GetTyped returns a typed version of the Component. Speaker implements types.Component
func (c Speaker) GetTyped() interface{} {
	return struct {
		Type string
		Speaker
	}{
		Type:    c.Type(),
		Speaker: c,
	}
}

//
// Intents
//

// SetSpeakerIntent represents an intent to change the speaker's state
type SetSpeakerIntent struct {
	OutputInPercent uint8
}

// Type returns IntentTypeSetSpeaker. SetSpeakerIntent implements types.Intent
func (i SetSpeakerIntent) Type() string { return IntentTypeSetSpeaker }

// GetTyped returns a typed version of the Intent. SetSpeakerIntent implements types.Intent
func (i SetSpeakerIntent) GetTyped() interface{} {
	return struct {
		Type string
		SetSpeakerIntent
	}{
		Type:             i.Type(),
		SetSpeakerIntent: i,
	}
}
