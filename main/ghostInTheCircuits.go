// +build ghost_in_the_circuits

package main

import (
	"github.com/fatih/color"
	"github.com/thejerf/suture"
	"math/rand"
	"sift"
	"sift/db"
	"sift/types"
	"time"
)

const (
	defaultMinMS = 5000
	defaultMaxMS = 7000
)

func main() {
	sift.SetLogLevel("crit")
	//ipv4.Log.SetHandler(log.DiscardHandler()) // ignore ipv4 scanner stuff

	// Instantiate a SIFT server
	server, err := sift.NewServer(sift.DefaultDBFilepath)
	if err != nil {
		panic(err)
	}
	if err = server.AddDefaults(); err != nil {
		panic(err)
	}

	// Start the server as a suture process
	supervisor := suture.NewSimple("repeat to console (SIFT app)")
	servToken := supervisor.Add(server)
	defer supervisor.Remove(servToken)
	go supervisor.ServeBackground()

	// Run the SIFT script
	ghostInTheCircuits(server, defaultMinMS, defaultMaxMS)
	select {}
}

// randTime produces a random duration between minMS and maxMS
func randTime(minMS, maxMS int) time.Duration {
	randDuration := maxMS - minMS
	if randDuration < 0 {
		randDuration = 10000
	}
	randMS := rand.Intn(randDuration)
	return time.Duration(minMS+randMS) * time.Millisecond
}

// This SIFT app will randomly alter any detected lights
func ghostInTheCircuits(server *sift.Server, minMS, maxMS int) {
	for {
		// Wait for a random amount of time
		<-time.After(randTime(minMS, maxMS))

		// Get all lights connected to the system
		lightsQuery := `
			SELECT c.name, c.device_id FROM component c
				JOIN device d ON c.device_id=d.id
				WHERE is_online=1 AND type=?`
		lights := []db.Component{}
		db, err := server.DB()
		if err != nil {
			color.Red("could not open DB: %v", err)
		}
		// run the query
		if err = db.Select(&lights, lightsQuery, types.LightEmitter{}.Type()); err != nil {
			color.Red("could not run query to get light ids: %v", err)
		}

		if len(lights) == 0 {
			color.Red("no lights found for the circuit ghosts to play with...")
		} else {
			// For each light found...
			for _, light := range lights {
				// ...assemble a component ID...
				lightID := types.ComponentID{
					Name:     light.Name,
					DeviceID: types.DeviceID(light.DeviceID),
				}

				// ...generate a random brightness value (0-100)...
				randBrightness := uint8(rand.Intn(100))

				// ...then create and submit an intent
				newLightIntent := types.SetLightEmitterIntent{
					BrightnessInPercent: randBrightness,
				}
				if err := server.EnactIntent(lightID, newLightIntent); err != nil {
					color.Red("could not enact intent: %v", err)
				} else {
					color.Blue("set light %v to %v", lightID, randBrightness)
				}
			}
		}
	}
}
