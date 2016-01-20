package sift_test

import (
	"fmt"
	"github.com/upwrd/sift"
	"github.com/upwrd/sift/adapter/example"
	"github.com/upwrd/sift/db"
	"github.com/upwrd/sift/notif"
	"github.com/upwrd/sift/types"
	. "gopkg.in/check.v1"
	"net"
	"strconv"
	"testing"
	"time"
)

// Hook up gocheck into the "go test" runner.
func TestSIFT(t *testing.T) { TestingT(t) }

const (
	testSiftSimplePort    = uint16(10952)
	testGetComponentsPort = uint16(10953)
	testSiftPort          = uint16(10954)
)

type SiftSuite struct{}

var _ = Suite(&SiftSuite{})

func (s *SiftSuite) TestServer(c *C) {
	// SetLogLevel("debug")

	// Start an example service server
	exServ := example.NewServer(testSiftPort)
	go exServ.Serve()
	defer exServ.Stop()
	<-time.After(500 * time.Millisecond)

	// Confirm that the example server is reachable
	url := "localhost:" + strconv.Itoa(int(testSiftPort))
	conn, err := net.DialTimeout("tcp", url, time.Second)
	c.Assert(err, IsNil)
	conn.Close()

	// start a new SIFT server
	siftServ, err := sift.NewServer("")
	c.Assert(err, IsNil)
	c.Assert(siftServ, NotNil)
	defer func() {
		err = siftServ.StopAndWait(1 * time.Minute)
		c.Assert(err, IsNil)
	}()
	go siftServ.Serve()
	components, err := siftServ.GetComponents(db.ExpandNone)
	c.Assert(err, IsNil)
	c.Assert(len(components), Equals, 0, Commentf("components: %+v", components))
	af := example.NewFactory(testSiftPort)
	_, err = siftServ.AddAdapterFactory(af)
	c.Assert(err, IsNil)

	// create a listener for component updates
	token := siftServ.Login()
	listener := siftServ.Listen(token, notif.ComponentFilter{}) // listen to components
	c.Assert(listener, NotNil)

	//
	// example --> SIFT
	//

	// Add a light to the example service
	light1 := example.Light{
		IsOn:            true,
		OutputInPercent: 100,
	}
	device1 := example.Device{
		Components: map[string]example.Component{"light1": light1},
	}
	exServ.SetDevice("device1", device1) // This should produce an update

	// Confirm that the SIFT server receives an update for the new light
	timeout := time.After(20 * time.Second)
	select {
	case <-listener:
	case <-timeout:
		c.Fatalf("SIFT timed out waiting for update from example server")
	}

	// The SIFT server should have one component, the LightEmitter
	components, err = siftServ.GetComponents(db.ExpandNone)
	c.Assert(err, IsNil)
	c.Assert(len(components), Equals, 1)
	var componentID types.ComponentID
	var component types.Component
	for key, comp := range components {
		component = comp
		componentID = key
	}
	c.Assert(componentID.Name, Not(Equals), "")
	c.Assert(componentID.DeviceID, Not(Equals), 0)
	expected := types.LightEmitter{
		BaseComponent: types.BaseComponent{
			Make:  "example",
			Model: "light_emitter_1",
		},
		State: types.LightEmitterState{
			BrightnessInPercent: 100,
		},
	}
	c.Assert(component, Equals, expected)

	//
	// SIFT --> example
	//

	// Send an intent to change the brightness of the light
	setBrightness := types.SetLightEmitterIntent{
		BrightnessInPercent: 42,
	}
	err = siftServ.EnactIntent(componentID, setBrightness)
	c.Assert(err, IsNil)

	timeout = time.After(20 * time.Second)
	select {
	case <-listener:
	case <-timeout:
		c.Fatalf("SIFT timed out waiting for update from example server")
	}

	components, err = siftServ.GetComponents(db.ExpandNone)
	c.Assert(err, IsNil)
	component, ok := components[componentID]
	c.Assert(ok, Equals, true)
	expected.State.BrightnessInPercent = 42 // updated expected to reflect server-side change
	c.Assert(component, Equals, expected)

	//	err = exServ.close()
	//	c.Assert(err, IsNil)
}

func Example() {
	// start a new SIFT server
	serv, _ := sift.NewServer("") // "" indicates a random, temporary file
	go serv.Serve()
	defer func() {
		serv.StopAndWait(1 * time.Minute)
	}()

	// With no running adapters, the SIFT server should not report any connected components
	components, _ := serv.GetComponents(db.ExpandAll)
	fmt.Printf("len(components) == %v", len(components)) // "len(components) == 0"

	// Start an example service server running on localhost port 12345. This is
	// a demonstration server that you can control programatically for testing.
	exServ := example.NewServer(uint16(12345))
	go exServ.Serve()
	defer exServ.Stop()
	<-time.After(500 * time.Millisecond)

	// Add a light to the example service. This code is specific to the example
	// server. In a real-world application, a light would be added when it is
	// plugged in and connected as defined by the manufacturer.
	light1 := example.Light{
		IsOn:            true,
		OutputInPercent: 100,
	}
	device1 := example.Device{
		Components: map[string]example.Component{"light1": light1},
	}
	exServ.SetDevice("device1", device1) // This should produce an update

	// Add an Adapter Factory to monitor the example service
	serv.AddAdapterFactory(example.NewFactory(12345))

	components, _ = serv.GetComponents(db.ExpandAll)
	fmt.Printf("len(components) == %v", len(components)) // "len(components) == 1"
	fmt.Printf("components:")
	for id, component := range components {
		fmt.Printf("   component %v: %+v", id, component)
	}
}
