package example

//import (
//	//"github.com/upwrd/sift/logging"
//	"fmt"
//	. "gopkg.in/check.v1"
//	"github.com/upwrd/sift/drivers"
//	"github.com/upwrd/sift/types"
//	"testing"
//	"time"
//)
//
//// Hook up gocheck into the "go test" runner.
//func TestSIFTExample(t *testing.T) { TestingT(t) }
//
//type TestSIFTExampleSuite struct{}
//
//var _ = Suite(&TestSIFTExampleSuite{})
//
//func (s *TestSIFTExampleSuite) TestIPv4Description(c *C) {
//	driver, err := Driver(DefaultPort)
//	c.Assert(err, IsNil)
//	c.Assert(driver, NotNil)
//	desc := driver.Services().IPv4
//	c.Assert(len(desc.OpenPorts), Equals, 1) // Only one port used by the example server
//	for i, _ := range desc.OpenPorts {
//		c.Assert(desc.OpenPorts[i], Equals, DefaultPort) // Default port should match
//	}
//}
//
//func pullOrTimeout(input chan interface{}) (interface{}, error) {
//	select {
//	case val := <-input:
//		return val, nil
//	case <-time.After(500 * time.Millisecond):
//		return nil, fmt.Errorf("timed out")
//	}
//}
//
//func (s *TestSIFTExampleSuite) TestIPv4ServerToDriverSync(c *C) {
//	//logging.SetLevelStr("debug")
//	driver, err := Driver(DefaultPort)
//	c.Assert(err, IsNil)
//	c.Assert(driver, NotNil)
//
//	go driver.Serve()
//	defer driver.Stop()
//
//	err = driver.SetOutput(nil) // cannot set output to nil channel
//	c.Assert(err, ErrorMatches, "destination channel cannot be nil")
//
//	updates := make(chan interface{}, 10)
//	err = driver.SetOutput(updates) // valid output, no error
//	c.Assert(err, IsNil)
//
//	err = driver.SetOutput(updates) // output already set, error
//	c.Assert(err, NotNil)
//
//	serv, shttp := StartServerWithHTTPTest(c)
//	c.Assert(serv, NotNil)
//	c.Assert(shttp, NotNil)
//	StartDriverAdaptingHTTP(c, driver, shttp)
//
//	// No devices should be produced, yet
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//	c.Assert(len(updates), Equals, 0)
//
//	// Emulate a light being added on the backend
//	light1 := Light{
//		BaseComponent:   BaseComponent{Type: ComponentTypeLight},
//		IsOn:            true,
//		OutputInPercent: 100,
//	}
//	device1 := Device{
//		Components: map[string]Component{"light 1": light1},
//	}
//	serv.SetDevice("device 1", device1) // This should produce an update
//
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//
//	c.Assert(len(updates), Equals, 1)
//	update, err := pullOrTimeout(updates)
//	c.Assert(err, IsNil)
//	c.Assert(update, NotNil)
//	device1SiftForm := convertDevice(device1)
//	var expected interface{}
//	expected = drivers.DeviceUpdated{
//		Key:       "device 1",
//		NewState: device1SiftForm,
//	}
//	c.Assert(update, DeepEquals, expected)
//
//	// Emulate updating the same light
//	light1State2 := Light{
//		BaseComponent:   BaseComponent{Type: ComponentTypeLight},
//		IsOn:            true,
//		OutputInPercent: 42,
//	}
//	device1State2 := Device{
//		Components: map[string]Component{"light 1": light1State2},
//	}
//	serv.SetDevice("device 1", device1State2) // This should produce an update
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//
//	c.Assert(len(updates), Equals, 1)
//	update, err = pullOrTimeout(updates)
//	c.Assert(err, IsNil)
//	c.Assert(update, NotNil)
//	device1State2SiftForm := convertDevice(device1State2)
//	expected = drivers.DeviceUpdated{
//		Key:       "device 1",
//		NewState: device1State2SiftForm,
//	}
//	c.Assert(update, DeepEquals, expected)
//
//	// Force a refresh - nothing should happen (since the devices haven't changed)
//	for i := 0; i < 10; i++ {
//		driver.ForceRefreshAll()
//	}
//	<-time.After(100 * time.Millisecond)
//	c.Assert(len(updates), Equals, 0)
//
//	// Delete device 1 from the server side
//	serv.RemoveDevice("device 1")
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//	c.Assert(len(updates), Equals, 1)
//	update, err = pullOrTimeout(updates)
//	c.Assert(err, IsNil)
//	c.Assert(update, NotNil)
//	expected = drivers.DeviceDeleted{
//		Key: "device 1",
//	}
//	c.Assert(update, DeepEquals, expected)
//}
//
//func (s *TestSIFTExampleSuite) TestIPv4ServerToDriverWithMultipleAdapters(c *C) {
//	//logging.SetLevelStr("debug")
//	driver, err := Driver(DefaultPort)
//	c.Assert(err, IsNil)
//	c.Assert(driver, NotNil)
//	go driver.Serve()
//	defer driver.Stop()
//
//	updates := make(chan interface{}, 10)
//	err = driver.SetOutput(updates)
//	c.Assert(err, IsNil)
//
//	serv, shttp := StartServerWithHTTPTest(c)
//	c.Assert(serv, NotNil)
//	c.Assert(shttp, NotNil)
//	StartDriverAdaptingHTTP(c, driver, shttp)
//
//	serv2, shttp2 := StartServerWithHTTPTest(c)
//	c.Assert(serv2, NotNil)
//	c.Assert(shttp2, NotNil)
//	StartDriverAdaptingHTTP(c, driver, shttp2)
//
//	// No devices should be produced, yet
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//	c.Assert(len(updates), Equals, 0)
//
//	// Emulate a light being added on the backend
//	light1 := Light{
//		BaseComponent:   BaseComponent{Type: ComponentTypeLight},
//		IsOn:            true,
//		OutputInPercent: 100,
//	}
//	device1 := Device{
//		Components: map[string]Component{"light 1": light1},
//	}
//	serv.SetDevice("device 1", device1) // This should produce an update
//
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//
//	c.Assert(len(updates), Equals, 1)
//	update, err := pullOrTimeout(updates)
//	c.Assert(err, IsNil)
//	c.Assert(update, NotNil)
//	device1SiftForm := convertDevice(device1)
//	var expected interface{}
//	expected = drivers.DeviceUpdated{
//		Key:       "device 1",
//		NewState: device1SiftForm,
//	}
//	c.Assert(update, DeepEquals, expected)
//
//	// Add device 1 to serv2. Since serv1 is higher priority than serv2 (added first),
//	// the update from serv2 should not be passed on by the Driver
//	serv2.SetDevice("device 1", device1)
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//	c.Assert(len(updates), Equals, 0)
//
//	// Delete device 1 from serv
//	serv.RemoveDevice("device 1")
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//	c.Assert(len(updates), Equals, 1)
//
//	expected = drivers.DeviceDeleted{
//		Key: "device 1",
//	}
//	update, err = pullOrTimeout(updates)
//	c.Assert(err, IsNil)
//	c.Assert(update, NotNil)
//	c.Assert(update, Equals, expected)
//
//	// Resubmit device 1 to serv2, unchanged. Since this is not a material change, adapter2 will
//	// not submit an update, and the Driver should not be notified
//	c.Assert(len(updates), Equals, 0)
//	serv2.SetDevice("device 1", device1)
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//	c.Assert(len(updates), Equals, 0)
//
//	// Submit a modified version of device 1 to serv2. Since serv has been removed, serv2's updates should now
//	// cause an update to be registered at the Driver level.
//	light1v2 := Light{
//		BaseComponent:   BaseComponent{Type: ComponentTypeLight},
//		IsOn:            true,
//		OutputInPercent: 42,
//	}
//	device1v2 := Device{
//		Components: map[string]Component{"light 1": light1v2},
//	}
//	c.Assert(len(updates), Equals, 0)
//	serv2.SetDevice("device 1", device1v2)
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//	c.Assert(len(updates), Equals, 1)
//}
//
//func (s *TestSIFTExampleSuite) TestIPv4AdapterToServer(c *C) {
//	//logging.SetLevelStr("debug")
//	driver, err := Driver(DefaultPort)
//	c.Assert(err, IsNil)
//	c.Assert(driver, NotNil)
//	go driver.Serve()
//	defer driver.Stop()
//
//	updates := make(chan interface{}, 10)
//	err = driver.SetOutput(updates)
//	c.Assert(err, IsNil)
//
//	serv, shttp := StartServerWithHTTPTest(c)
//	c.Assert(serv, NotNil)
//	c.Assert(shttp, NotNil)
//	StartDriverAdaptingHTTP(c, driver, shttp)
//
//	// Emulate a light being added on the backend
//	light1 := Light{
//		BaseComponent:   BaseComponent{Type: ComponentTypeLight},
//		IsOn:            true,
//		OutputInPercent: 100,
//	}
//	device1 := Device{
//		Components: map[string]Component{"light1": light1},
//	}
//	serv.SetDevice("device1", device1) // This should produce an update
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//
//	c.Assert(len(updates), Equals, 1)
//	update, err := pullOrTimeout(updates)
//	c.Assert(err, IsNil)
//	c.Assert(update, NotNil)
//	device1SiftForm := convertDevice(device1)
//	var expected interface{}
//	expected = drivers.DeviceUpdated{
//		Key:       "device1",
//		NewState: device1SiftForm,
//	}
//	c.Assert(update, DeepEquals, expected)
//
//	// Have the Adapter turn the light down
//	c.Assert(len(updates), Equals, 0)
//	updateComponentIntent := types.SetLightEmitterIntent{
//		BrightnessInPercent: uint8(42),
//	}
//
//	err = driver.EnactIntent("device1", "light1", updateComponentIntent)
//	c.Assert(err, IsNil)
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//
//	c.Assert(len(updates), Equals, 1)
//	update, err = pullOrTimeout(updates)
//	c.Assert(err, IsNil)
//	c.Assert(update, NotNil)
//
//	light1State2 := Light{
//		BaseComponent:   BaseComponent{Type: ComponentTypeLight},
//		IsOn:            true,
//		OutputInPercent: 42,
//	}
//	device1State2 := Device{
//		Components: map[string]Component{"light1": light1State2},
//	}
//	device1State2SiftForm := convertDevice(device1State2)
//	expected = drivers.DeviceUpdated{
//		Key:       "device1",
//		NewState: device1State2SiftForm,
//	}
//	c.Assert(update, DeepEquals, expected)
//}
//
//func (s *TestSIFTExampleSuite) TestIPv4DriverToServerWithMultipleAdapters(c *C) {
//	//logging.SetLevelStr("debug")
//	driver, err := Driver(DefaultPort)
//	c.Assert(err, IsNil)
//	c.Assert(driver, NotNil)
//	go driver.Serve()
//	defer driver.Stop()
//
//	updates := make(chan interface{}, 10)
//	err = driver.SetOutput(updates)
//	c.Assert(err, IsNil)
//
//	serv, shttp := StartServerWithHTTPTest(c)
//	c.Assert(serv, NotNil)
//	c.Assert(shttp, NotNil)
//	StartDriverAdaptingHTTP(c, driver, shttp)
//
//	serv2, shttp2 := StartServerWithHTTPTest(c)
//	c.Assert(serv2, NotNil)
//	c.Assert(shttp2, NotNil)
//	StartDriverAdaptingHTTP(c, driver, shttp2)
//
//	// No devices should be produced, yet
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//	c.Assert(len(updates), Equals, 0)
//
//	// Emulate a light being added on the backend
//	light1 := Light{
//		BaseComponent:   BaseComponent{Type: ComponentTypeLight},
//		IsOn:            true,
//		OutputInPercent: 100,
//	}
//	device1 := Device{
//		Components: map[string]Component{"light 1": light1},
//	}
//	serv.SetDevice("device 1", device1) // This should produce an update
//
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//
//	c.Assert(len(updates), Equals, 1)
//	update, err := pullOrTimeout(updates)
//	c.Assert(err, IsNil)
//	c.Assert(update, NotNil)
//	device1SiftForm := convertDevice(device1)
//	var expected interface{}
//	expected = drivers.DeviceUpdated{
//		Key:       "device 1",
//		NewState: device1SiftForm,
//	}
//	c.Assert(update, DeepEquals, expected)
//
//	// Add device 1 to serv2. Since serv1 is higher priority than serv2 (added first),
//	// the update from serv2 should not be passed on by the Driver
//	serv2.SetDevice("device 1", device1)
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//	c.Assert(len(updates), Equals, 0)
//
//	// Have adapter turn the light down
//	c.Assert(len(updates), Equals, 0)
//	updateComponentIntent := types.SetLightEmitterIntent{
//		BrightnessInPercent: uint8(42),
//	}
//
//	err = driver.EnactIntent("device 1", "light 1", updateComponentIntent)
//	c.Assert(err, IsNil)
//	driver.ForceRefreshAll() // Since light 1 was changed, the server should indicate an update
//	<-time.After(100 * time.Millisecond)
//	c.Assert(len(updates), Equals, 1)
//	update, err = pullOrTimeout(updates)
//	c.Assert(err, IsNil)
//	c.Assert(update, NotNil)
//	light1State2 := Light{
//		BaseComponent:   BaseComponent{Type: ComponentTypeLight},
//		IsOn:            true,
//		OutputInPercent: 42,
//	}
//	device1State2 := Device{
//		Components: map[string]Component{"light 1": light1State2},
//	}
//	device1State2SiftForm := convertDevice(device1State2)
//	expected = drivers.DeviceUpdated{
//		Key:       "device 1",
//		NewState: device1State2SiftForm,
//	}
//	c.Assert(update, DeepEquals, expected)
//
//	// Delete device 1 from serv
//	serv.RemoveDevice("device 1")
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//	c.Assert(len(updates), Equals, 1)
//
//	expected = drivers.DeviceDeleted{
//		Key: "device 1",
//	}
//	update, err = pullOrTimeout(updates)
//	c.Assert(err, IsNil)
//	c.Assert(update, NotNil)
//	c.Assert(update, Equals, expected)
//
//	// Have the driver change the light again. This should be handled by adapter2
//	c.Assert(len(updates), Equals, 0)
//	updateComponentIntent2 := types.SetLightEmitterIntent{
//		BrightnessInPercent: uint8(77),
//	}
//	err = driver.EnactIntent("device 1", "light 1", updateComponentIntent2)
//	c.Assert(err, IsNil)
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//	c.Assert(len(updates), Equals, 1)
//	update, err = pullOrTimeout(updates)
//	c.Assert(err, IsNil)
//	c.Assert(update, NotNil)
//	light1State3 := Light{
//		BaseComponent:   BaseComponent{Type: ComponentTypeLight},
//		IsOn:            true,
//		OutputInPercent: 77,
//	}
//	device1State3 := Device{
//		Components: map[string]Component{"light 1": light1State3},
//	}
//	device1State3SiftForm := convertDevice(device1State3)
//	expected = drivers.DeviceUpdated{
//		Key:       "device 1",
//		NewState: device1State3SiftForm,
//	}
//	c.Assert(update, DeepEquals, expected)
//
//	// Delete device 1 from serv2
//	serv2.RemoveDevice("device 1")
//	driver.ForceRefreshAll()
//	<-time.After(100 * time.Millisecond)
//	c.Assert(len(updates), Equals, 1)
//
//	expected = drivers.DeviceDeleted{
//		Key: "device 1",
//	}
//	update, err = pullOrTimeout(updates)
//	c.Assert(err, IsNil)
//	c.Assert(update, NotNil)
//	c.Assert(update, Equals, expected)
//
//	// Try changing device 1. Since both of the Adapters have reported it as "deleted",
//	// the Driver should produce an error
//	err = driver.EnactIntent("device 1", "light 1", updateComponentIntent)
//	c.Assert(err, ErrorMatches, "device 1 is not being handled by this driver.*")
//}
