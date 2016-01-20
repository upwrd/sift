package example

import (
	"encoding/json"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Hook up gocheck into the "go test" runner.
func TestSIFTExample(t *testing.T) { TestingT(t) }

type TestSIFTExampleSuite struct{}

var _ = Suite(&TestSIFTExampleSuite{})

func (s *TestSIFTExampleSuite) TestStatus(c *C) {
	server := NewServer(testPort)
	c.Assert(server, NotNil)

	router := server.Handlers("")
	ts := httptest.NewUnstartedServer(router)
	c.Assert(ts, NotNil)
	ts.Config.Addr = ":10203"
	ts.Start()
	defer ts.Close()

	status := getStatus(ts, c)
	c.Assert(status.Type, Equals, serverTypeAllAtOnce)
}

func (s *TestSIFTExampleSuite) TestGetDevices(c *C) {
	server := NewServer(testPort) // Start a server
	c.Assert(server, NotNil)
	router := server.Handlers("")
	c.Assert(router, NotNil)
	ts := httptest.NewServer(router)
	c.Assert(ts, NotNil)
	defer ts.Close()
	devices := getDevices(ts, c)
	c.Assert(len(devices), Equals, 0) // Should be no devices on the server yet

	// Insert a device
	light1 := Light{
		IsOn:            true,
		OutputInPercent: 100,
	}
	device1 := Device{
		Components: map[string]Component{"light 1": light1},
	}
	server.SetDevice("device 1", device1)
	devices = getDevices(ts, c) // Get all Devices from the server
	c.Assert(len(devices), Equals, 1)
	c.Assert(devices["device 1"], DeepEquals, device1) // Server should have device1
	device1FromServer := getDevice(ts, c, "device 1")
	c.Assert(device1FromServer, DeepEquals, device1) // Server should have device1

	// Update the device
	light1State2 := Light{
		IsOn:            false,
		OutputInPercent: 42,
	}
	device1State2 := Device{
		Components: map[string]Component{"light 1": light1State2},
	}
	server.SetDevice("device 1", device1State2)
	devices = getDevices(ts, c)
	c.Assert(len(devices), Equals, 1)
	c.Assert(devices["device 1"], DeepEquals, device1State2) // Server should have updated device1
	device1FromServer = getDevice(ts, c, "device 1")
	c.Assert(device1FromServer, DeepEquals, device1State2) // Server should have device1

	// Remove the device
	server.removeDevice("device 1")
	devices = getDevices(ts, c)
	c.Assert(len(devices), Equals, 0)
}

func (s *TestSIFTExampleSuite) TestNotifications(c *C) {
	server := NewServer(testPort) // Instantiate a server
	go server.Serve()             // Server needs to be running to post notifications
	c.Assert(server, NotNil)
	router := server.Handlers("")
	c.Assert(router, NotNil)
	ts := httptest.NewServer(router)
	c.Assert(ts, NotNil)
	defer ts.Close()
	devices := getDevices(ts, c)
	c.Assert(len(devices), Equals, 0) // Should be no devices on the server yet

	hasChanged := make(chan bool, 10) // hasChanged will listen for changes
	server.listenForChanges(hasChanged)
	c.Assert(len(hasChanged), Equals, 0)

	// Insert a device
	light1 := Light{
		IsOn:            true,
		OutputInPercent: 100,
	}
	device1 := Device{
		Components: map[string]Component{"light 1": light1},
	}
	server.SetDevice("device 1", device1)
	devices = getDevices(ts, c) // Get all Devices from the server
	c.Assert(len(devices), Equals, 1)
	c.Assert(devices["device 1"], DeepEquals, device1) // Server should have device1
	time.Sleep(100 * time.Millisecond)
	c.Assert(len(hasChanged), Equals, 1) // Confirm that a notification was sent

	// Update the device
	light1State2 := Light{
		IsOn:            false,
		OutputInPercent: 42,
	}
	device1State2 := Device{
		Components: map[string]Component{"light 1": light1State2},
	}
	server.SetDevice("device 1", device1State2)
	devices = getDevices(ts, c)
	c.Assert(len(devices), Equals, 1)
	c.Assert(devices["device 1"], DeepEquals, device1State2) // Server should have updated device1
	time.Sleep(100 * time.Millisecond)
	c.Assert(len(hasChanged), Equals, 2) // Confirm that a notification was sent

	// Remove the device
	server.removeDevice("device 1")
	devices = getDevices(ts, c)
	c.Assert(len(devices), Equals, 0)
	time.Sleep(100 * time.Millisecond)
	c.Assert(len(hasChanged), Equals, 3) // Confirm that a notification was sent
}

func (s *TestSIFTExampleSuite) TestDevicePOST(c *C) {
	server := NewServer(testPort) // Start a server
	c.Assert(server, NotNil)
	router := server.Handlers("")
	c.Assert(router, NotNil)
	ts := httptest.NewServer(router)
	c.Assert(ts, NotNil)
	defer ts.Close()
	devices := getDevices(ts, c)
	c.Assert(len(devices), Equals, 0) // Should be no devices on the server yet

	// Insert a device
	light1 := Light{
		IsOn:            true,
		OutputInPercent: 100,
	}
	device1 := Device{
		Components: map[string]Component{"light 1": light1},
	}
	device1FromPost := postDevice(ts, c, "device 1", device1)
	c.Assert(device1FromPost, DeepEquals, device1)
	devices = getDevices(ts, c) // Get all Devices from the server
	c.Assert(len(devices), Equals, 1)
	c.Assert(devices["device 1"], DeepEquals, device1) // Server should have device1

	// Update the device
	light1State2 := Light{
		IsOn:            false,
		OutputInPercent: 42,
	}
	device1State2 := Device{
		Components: map[string]Component{"light 1": light1State2},
	}
	device1State2FromPost := postDevice(ts, c, "device 1", device1State2)
	c.Assert(device1State2FromPost, DeepEquals, device1State2)
	devices = getDevices(ts, c)
	c.Assert(len(devices), Equals, 1)
	c.Assert(devices["device 1"], DeepEquals, device1State2) // Server should have updated device1
}

func (s *TestSIFTExampleSuite) TestComponentGETandPOST(c *C) {
	server := NewServer(testPort) // Start a server
	c.Assert(server, NotNil)
	router := server.Handlers("")
	c.Assert(router, NotNil)
	ts := httptest.NewServer(router)
	c.Assert(ts, NotNil)
	defer ts.Close()
	devices := getDevices(ts, c)
	c.Assert(len(devices), Equals, 0) // Should be no devices on the server yet

	// Insert a device
	light1 := Light{
		IsOn:            true,
		OutputInPercent: 100,
	}
	device1 := Device{
		Components: map[string]Component{"light 1": light1},
	}
	device1FromPost := postDevice(ts, c, "device 1", device1)
	c.Assert(device1FromPost, DeepEquals, device1)
	devices = getDevices(ts, c) // Get all Devices from the server
	c.Assert(len(devices), Equals, 1)
	c.Assert(devices["device 1"], DeepEquals, device1) // Server should have device1

	light1FromCompPath := getComponent(ts, c, "device 1", "light 1")
	c.Assert(light1FromCompPath, DeepEquals, light1)

	// Update the component individually
	light1State2 := Light{
		IsOn:            false,
		OutputInPercent: 42,
	}
	device1State2 := Device{
		Components: map[string]Component{"light 1": light1State2},
	}
	light1State2FromCompPath := postComponent(ts, c, "device 1", "light 1", light1State2)
	c.Assert(light1State2FromCompPath, Equals, light1State2)

	devices = getDevices(ts, c) // Get all Devices from the server
	c.Assert(len(devices), Equals, 1)
	c.Assert(devices["device 1"], DeepEquals, device1State2)
}

//
// JSON tests
//
var componentTests = map[string]Component{
	"empty light": Light{},
	"light": Light{
		IsOn:            true,
		OutputInPercent: uint8(42),
	},
	"lock": Lock{
		IsOpen: true,
	},
}

func (s *TestSIFTExampleSuite) TestComponentJSONBackAndForth(c *C) {
	for testName, testComponent := range componentTests {
		asJSON, err := json.Marshal(testComponent)
		c.Assert(err, IsNil, Commentf("failed case: %v", testName))
		remarshalled, err := componentFromJSON(asJSON)
		c.Assert(err, IsNil, Commentf("failed case: %v", testName))
		c.Assert(remarshalled, DeepEquals, testComponent, Commentf("failed case: %v", testName))
	}
}

var deviceTests = map[string]Device{
	"empty device": Device{Components: map[string]Component{}},
	"device with light": Device{
		Components: map[string]Component{
			"light": Light{
				IsOn:            true,
				OutputInPercent: uint8(42),
			},
		},
	},
	"device with light and lock": Device{
		Components: map[string]Component{
			"light": Light{
				IsOn:            true,
				OutputInPercent: uint8(42),
			},
			"lock": Lock{
				IsOpen: true,
			},
		},
	},
}

func (s *TestSIFTExampleSuite) TestDeviceJSONBackAndForth(c *C) {
	for testName, testDevice := range deviceTests {
		asJSON, err := json.Marshal(testDevice)
		c.Assert(err, IsNil, Commentf("failed case: %v", testName))
		remarshalled, err := deviceFromJSON(asJSON)
		c.Assert(err, IsNil, Commentf("failed case: %v", testName))
		c.Assert(remarshalled, DeepEquals, testDevice, Commentf("failed case: %v", testName))
	}
}

//
// Helper functions
//

func getDevices(ts *httptest.Server, c *C) map[string]Device {
	res, err := http.Get(ts.URL + "/devices")
	c.Assert(err, IsNil)
	devicesJSON, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	c.Assert(err, IsNil)

	devices, err := devicesFromJSON(devicesJSON)
	c.Assert(err, IsNil)
	return devices
}

func getDevice(ts *httptest.Server, c *C, id string) Device {
	res, err := http.Get(ts.URL + "/devices/" + id)
	c.Assert(err, IsNil)
	deviceJSON, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	c.Assert(err, IsNil)

	device, err := deviceFromJSON(deviceJSON)
	c.Assert(err, IsNil, Commentf("tried parsing GET response: %s", deviceJSON))
	return device
}

func postDevice(ts *httptest.Server, c *C, id string, d Device) Device {
	asJSON, err := json.Marshal(d) // convert Device to JSON
	c.Assert(err, IsNil)
	res, err := http.Post(ts.URL+"/devices/"+id, "Application/JSON", strings.NewReader(string(asJSON))) // do POST
	respBody, err := ioutil.ReadAll(res.Body)
	c.Assert(err, IsNil)
	res.Body.Close()
	deviceFromResponse, err := deviceFromJSON(respBody)
	c.Assert(err, IsNil, Commentf("tried parsing POST response: %s", respBody))
	return deviceFromResponse
}

func getComponent(ts *httptest.Server, c *C, devID, compID string) Component {
	res, err := http.Get(ts.URL + "/devices/" + devID + "/" + compID)
	c.Assert(err, IsNil)
	compJSON, err := ioutil.ReadAll(res.Body)
	c.Assert(err, IsNil)
	res.Body.Close()

	comp, err := componentFromJSON(compJSON)
	c.Assert(err, IsNil, Commentf("tried parsing GET response: %s", compJSON))
	return comp
}

func postComponent(ts *httptest.Server, c *C, devID, compID string, comp Component) Component {
	asJSON, err := json.Marshal(comp) // convert Component to JSON
	c.Assert(err, IsNil)
	res, err := http.Post(ts.URL+"/devices/"+devID+"/"+compID, "Application/JSON", strings.NewReader(string(asJSON))) // do POST
	respBody, err := ioutil.ReadAll(res.Body)
	c.Assert(err, IsNil)
	res.Body.Close()
	compFromResponse, err := componentFromJSON(respBody)
	c.Assert(err, IsNil, Commentf("tried parsing POST response: %s", respBody))
	return compFromResponse
}

func getStatus(ts *httptest.Server, c *C) serverStatus {
	res, err := http.Get(ts.URL + "/status")
	c.Assert(err, IsNil)
	statusJSON, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	c.Assert(err, IsNil)

	var status serverStatus
	err = json.Unmarshal(statusJSON, &status)
	c.Assert(err, IsNil)
	return status
}
