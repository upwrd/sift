// +build example_server

package main

import (
	"fmt"
	//"github.com/thejerf/suture"
	"sift"
	"sift/drivers/example"
)

func main() {
	sift.SetLogLevel("info")
	server := example.NewServer(example.defaultPort)
	go server.Serve()
	//	supervisor := suture.NewSimple("SIFT example main")
	//	supervisor.Add(server)
	//	go supervisor.ServeBackground()
	// go server.serveHTTP()

	// Insert a device
	light1 := example.light{
		baseComponent:   example.baseComponent{Type: example.componentTypeLight},
		IsOn:            true,
		OutputInPercent: 100,
	}
	device1 := example.device{
		Components: map[string]example.component{"light 1": light1},
	}
	server.SetDevice("device 1", device1)

	fmt.Printf("SIFT example server started\n")
	select {} // wait forever
}
