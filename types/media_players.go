package types

// string constants
const (
	ComponentTypeMediaPlayer          = "media_player"
	IntentTypeSetMediaPlayerPlayState = "set_media_player_play_state"
)

// Possible media player states
const (
	MediaPlayerStateIdle      = "IDLE"
	MediaPlayerStateStopped   = "STOPPED"
	MediaPlayerStateBuffering = "BUFFERING"
	MediaPlayerStatePaused    = "PAUSED"
	MediaPlayerStatePlaying   = "PLAYING"
)

// Possible media types
const (
	MediaTypeVideo = "VIDEO"
	MediaTypeAudio = "AUDIO"
)

// MediaPlayer represents a real-world media player, like a light bulb or lamp.
type MediaPlayer struct {
	BaseComponent
	State MediaPlayerState
	Stats *MediaPlayerStats
	Specs *MediaPlayerSpecs
}

// MediaPlayerState represents the state of a real-world media player.
type MediaPlayerState struct {
	// IDLE, STOPPED, BUFFERING, PAUSED, PLAYING
	//PlayState string `db:"play_state",json:"play_state"`
	PlayState string `db:"play_state" json:"play_state"`

	// AUDIO, VIDEO
	MediaType string `db:"media_type" json:"media_type"`

	// YouTube, Netflix, Plex, etc.
	Source string
}

// MediaPlayerStats contains statistics about the media player.
type MediaPlayerStats struct {
	HoursOn int `db:"hours_on"`
}

// MediaPlayerSpecs represents the specifications of a real-world media player.
type MediaPlayerSpecs struct {
	SupportedAudioTypes string `db:"supported_audio_types" json:"supported_audio_types"`
	SupportedVideoTypes string `db:"supported_video_types" json:"supported_video_types"`
}

// Type returns ComponentTypeMediaPlayer. MediaPlayer implements types.Component
func (c MediaPlayer) Type() string { return ComponentTypeMediaPlayer }

// GetTyped returns a typed version of the Component. MediaPlayer implements types.Component
func (c MediaPlayer) GetTyped() interface{} {
	return struct {
		Type string
		MediaPlayer
	}{
		Type:        c.Type(),
		MediaPlayer: c,
	}
}

//
// Intents
//

// SetMediaPlayerIntent represents an intent to change the media player's state
type SetMediaPlayerIntent struct {
	PlayState string `db:"play_state" json:"play_state"`
}

// Type returns IntentTypeSetMediaPlayer. SetMediaPlayerIntent implements types.Intent
func (i SetMediaPlayerIntent) Type() string { return IntentTypeSetMediaPlayerPlayState }

// GetTyped returns a typed version of the Intent. SetMediaPlayerIntent implements types.Intent
func (i SetMediaPlayerIntent) GetTyped() interface{} {
	return struct {
		Type string
		SetMediaPlayerIntent
	}{
		Type:                 i.Type(),
		SetMediaPlayerIntent: i,
	}
}
