// +build repeat_to_console

package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/thejerf/suture"
	"sift"
	"sift/notif"

	// log "gopkg.in/inconshreveable/log15.v2"
	// "sift/network/ipv4"
)

func main() {
	sift.SetLogLevel("debug")
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
	repeatUpdatesToConsole(server)
}

// main starts a sift server and listens for updates, printing any to console.
func repeatUpdatesToConsole(server *sift.Server) {
	myToken := server.Login()
	updateChan := server.Listen(myToken) // without specifying filters, this will listen to everything

	fmt.Println("listening to SIFT server and printing updates to console...")
	for {
		update := <-updateChan //TODO: uncaught panic if updateChan is closed
		switch typed := update.(type) {
		case notif.ComponentNotification:
			color.Blue("component %+v %v: %+v\n", typed.ID, typed.Action, typed.Component)
		//case notif.DriverNotification:
		//	fmt.Printf("driver %v: %v", typed.NotificationType, typed.Component)
		default:
			color.Red("unhandled update type from updateChan: %T (%v)\n", update, update)
		}
	}
}
