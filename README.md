# SIFT (Simple Interface for Functional Things)

[![Godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/upwrd/sift) [![Build Status](https://travis-ci.org/upwrd/sift.svg?branch=master)](https://travis-ci.org/upwrd/sift)

SIFT is a framework that enables developers to control connected devices without
having to understand their implementation details.

A SIFT server:
* Regularly scans available networks to discover new connected devices
* Once discovered, actively gathers the state of connected devices to produce
  a synchronized internal collection of device states
* Allows developers to query the state of connected devices, and to subscribe
  to notifications when they are updated or removed
* Allows developers to control controllable devices

_Note: SIFT is in early development. The API may change in future updates._

## Getting Started

SIFT requires Golang 1.4+ and python 2.7.

```
# install
go get github.com/upwrd/sift
pip install -r $GOPATH/github.com/upwrd/sift/requirements.txt

# run interactive setup & example apps:
go run github.com/upward/sift/main/interactive.go
```

## Examples

#### Movies and Chill

Automatically turn down the lights when watching a movie

```go
package main

import (
  "github.com/upwrd/sift"
  "github.com/upwrd/sift/types"
  "github.com/upwrd/sift/notif"
)

func main() {
  // Create a SIFT server at the default path ("sift.db")
  server, _ := sift.NewServer(sift.DefaultDBFilepath) // err ignored
  server.AddDefaults() // err ignored
  go server.Serve() // starts discovering and syncing devices

  token := server.Login() // authenticate with SIFT

  // Subscribe to receive notifications when media players or lights are added
  // or removed
  mediaFilter := notif.ComponentFilter{Type: types.ComponentTypeMediaPlayer}
  lightsFilter := notif.ComponentFilter{Type: types.ComponentTypeLightEmitter}
  listener := server.Listen(token, lightsFilter, mediaFilter) // this starts listening

  // Each time a light or media player is added or changed...
  for range listener {
    // ...determine which lights should be dim and which should be bright...
    db, _ := server.DB()
    lightsQuery := `
    			SELECT c1.device_id, c1.name, num_playing_mpls_in_room
    			FROM light_emitter_state l1
    			JOIN component c1 ON l1.id=c1.id
    			JOIN device d1 ON c1.device_id=d1.id
    			JOIN (SELECT d.location_id loc,
    				SUM (CASE WHEN m.play_state="PLAYING" THEN 1 ELSE 0 END) as num_playing_mpls_in_room
    				FROM media_player_state m
    				JOIN component c ON m.id=c.id
    				JOIN device d ON c.device_id=d.id
    				GROUP BY d.location_id)
    				ON d1.location_id=loc;`
    type result struct {
    	DeviceID                     int64 `db:"device_id"`
    	Name                         string
    	NumPlayingMediaPlayersInRoom int `db:"num_playing_mpls_in_room"`
    }
    results := []result{}
    db.Select(&results, lightsQuery) // err ignored

    // ...then tell the SIFT server to make it so
  	for _, result := range results {
  		var intent types.SetLightEmitterIntent // SIFT uses intents to control things
  		if result.NumPlayingMediaPlayersInRoom > 0 { // a movie is playing in the room ...
  			intent.BrightnessInPercent = lightsLow // ...bring the lights down low
  		} else { // no movies in this room...
  			intent.BrightnessInPercent = lightsHigh // ...bring the lights up
  		}
  		target := types.ComponentID{
  			DeviceID: types.DeviceID(result.DeviceID),
  			Name:     result.Name,
  		}
  		// Send the intent to the SIFT server, which will try to make it real
  		server.EnactIntent(target, intent) // err ignored
    }
  }
}
```

## Data Model
To learn more about the SIFT data types and structure, see:
https://github.com/upwrd/sift/types

## Compatibility

| Device                    | Networks | Components     | Notes                                                                 |
|---------------------------|----------|----------------|-----------------------------------------------------------------------|
| Connected By TCP Lighting | IPv4     | Light Emitters | Must press the sync button on the Connected By TCP Hub on first run.                                                                      |
| Google Chromecast         | IPv4     | Media Players  | Does not work with all Chromecast apps, most notably  Netflix :( |

To learn how to add support for new devices and device types, see
https://github.com/upwrd/sift/drivers
