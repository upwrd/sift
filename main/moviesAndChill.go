// +build movies_and_chill

package main

import (
	"fmt"
	"github.com/thejerf/suture"
	"github.com/upwrd/sift"
	"github.com/upwrd/sift/notif"
	"github.com/upwrd/sift/types"
)

const (
	lightsLow  = 10  // percent
	lightsHigh = 100 // percent
)

func main() {
	// Instantiate a SIFT server
	server, err := sift.NewServer(sift.DefaultDBFilepath)
	if err != nil {
		panic(err)
	}
	if err = server.AddDefaults(); err != nil {
		panic(err)
	}

	// Start the server as a suture process
	supervisor := suture.NewSimple("movies and chill (SIFT app)")
	servToken := supervisor.Add(server)
	defer supervisor.Remove(servToken)
	go supervisor.ServeBackground()

	// Run the SIFT script
	moviesAndChill(server)
}

// This SIFT app will turn the lights down in any room with a running media
// player. If there are no running media players, the lights will be set to
// high.
func moviesAndChill(server *sift.Server) {
	if server == nil { // sanity check
		return
	}
	token := server.Login() // authenticate with SIFT

	// SIFT apps can request notifications from SIFT to determine when they
	// should do something. For this app, we'll want to know when media players
	// change (e.g. if they are added or their states change), and when lights
	// change (e.g. if they are added, moved, or removed)
	mediaFilter := notif.ComponentFilter{Type: types.ComponentTypeMediaPlayer}
	lightsFilter := notif.ComponentFilter{Type: types.ComponentTypeLightEmitter}
	listener := server.Listen(token, lightsFilter, mediaFilter) // this starts listening

	for range listener {
		// Each time the listener receives a notification, this example will
		// recalculate the lighting values for each light.
		//   (A better implementation might look at the updates and only
		//   recalculate those that need to be recalculated)

		// Run a query against the SIFT sqlite database to find each available
		// light and the number of PLAYING media players in the same room.
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
		db, _ := server.DB()
		// run the query
		if err := db.Select(&results, lightsQuery); err != nil {
			panic(fmt.Sprintf("could not run query to get active media players: %v", err))
		}

		// Check out the results and determine how each light should be set. In
		// SIFT, this is done using Intents.
		for _, result := range results {
			var intent types.SetLightEmitterIntent
			if result.NumPlayingMediaPlayersInRoom > 0 { // a movie is playing in the room ...
				intent.BrightnessInPercent = lightsLow // ...bring the lights down low
			} else { // no movies in this room...
				intent.BrightnessInPercent = lightsHigh // ...bring the lights up
			}
			target := types.ComponentID{
				DeviceID: types.DeviceID(result.DeviceID),
				Name:     result.Name,
			}
			// Send the intent to the SIFT server, which will make it real
			if err := server.EnactIntent(target, intent); err != nil {
				fmt.Printf("warning: could not enact intent: %v\n", err)
			}
		}
	}
}
