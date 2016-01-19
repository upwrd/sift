package example

import (
	"encoding/json"
	"fmt"
	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"
	"io/ioutil"
	"net"
	"net/http"
	"sift/lib"
	"sift/logging"
	"sift/network/ipv4"
	"sift/types"
	"strconv"
	"strings"
	"time"
	"sift/drivers"
)

// Log is used to log messages for the example package. Logs are disabled by
// default; use sift/logging.SetLevel() to set log levels for all packages, or
// Log.SetHandler() to set a custom handler for this package (see:
// https://godoc.org/gopkg.in/inconshreveable/log15.v2)
var Log = logging.Log.New("pkg", "drivers/example")

const (
	timeBetweenPolls           = 5 * time.Second
	timeBetweenHeartbeats      = 5 * time.Second
	serverCommunicationTimeout = 5 * time.Second
)

// Constant values for the example light specs. These are totally made up, and
// are exported for testing purposes.
const (
	LampMaxOutputInLumens       = 700
	LampMinOutputInLumens       = 0
	LampExpectedLifetimeInHours = 10000
)

const (
	manufacturer = "example"
)

// An AdapterFactory creates adapters
type AdapterFactory struct {
	port uint16
}

// NewFactory properly instantiates a new AdapterFactory
func NewFactory(port uint16) *AdapterFactory {
	return &AdapterFactory{
		port: port,
	}
}

// HandleIPv4 spawns a new Adapter to handle a context
func (f *AdapterFactory) HandleIPv4(context ipv4.ServiceContext) drivers.Adapter {
	return newAdapter(f.port, context)
}

// GetIPv4Description returns a description of the example IPv4 service that
// can be used to identify example services on a network
func (f *AdapterFactory) GetIPv4Description() ipv4.ServiceDescription {
	return ipv4.ServiceDescription{OpenPorts: []uint16{f.port}}
}

// Name returns the name of this adapter factory, "SIFT example"
func (f *AdapterFactory) Name() string { return "SIFT example" }

type ipv4Adapter struct {
	port             uint16
	updateChan       chan interface{}
	context          ipv4.ServiceContext
	differ           lib.SetOutputBasedDeviceDiffer
	desc             lib.AdapterDescription
	stop             chan struct{}
	debgForceRefresh chan struct{}
	log              log.Logger
}

func newAdapter(port uint16, context ipv4.ServiceContext) *ipv4Adapter {
	log := Log.New("obj", "example ipv4 adapter", "id", logext.RandId(8), "adapting", context.IP.String())
	log.Info("example adapter created")
	adapter := &ipv4Adapter{
		port:             port,
		updateChan:       make(chan interface{}, 100),
		context:          context,
		differ:           lib.NewAllAtOnceDiffer(),
		stop:             make(chan struct{}),
		debgForceRefresh: make(chan struct{}, 10),
		log:              log,
	}
	if err := adapter.differ.SetOutput(adapter.updateChan); err != nil {
		panic(fmt.Sprintf("newAdapter() could not set output: %v", err))
	}
	go adapter.Serve()
	return adapter
}

// UpdateChan returns a channel which will be populated with updates from the
// adapter
func (a *ipv4Adapter) UpdateChan() chan interface{} {
	return a.updateChan
}

// Serve begins adapting the example service specified by the adapter's
// context. As updates within the service are found, they will be sent to the
// update channel provided by UpdateChan(). While the adapter is serving,
// heartbeat messages will be sent to the adapter's context's status channel.
func (a *ipv4Adapter) Serve() {
	// Check if the ipv4 context that we were given represents an example service
	if !a.isExampleService(a.context) {
		a.log.Info("%s was not an example service\n", a.context.IP.String())
		a.context.SendStatus(ipv4.DriverStatusIncorrectService)
		return
	}

	if a.differ == nil {
		a.log.Warn("example ipv4 Adapter was improperly instantiated!")
		a.context.SendStatus(ipv4.DriverStatusError)
		return
	}

	// Send heartbeats to the caller as long as this service is being handled.
	stopHeartbeating := make(chan struct{})
	defer func() {
		stopHeartbeating <- struct{}{}
	}()
	go func() {
		heartbeat := time.NewTimer(0)
		for {
			select {
			case <-stopHeartbeating:
				return
			case <-heartbeat.C:
				// Try to send a heartbeat status
				if err := a.context.SendStatus(ipv4.DriverStatusHandling); err != nil {
					return // Context must have been killed, stop heartbeating
				}
				heartbeat.Reset(timeBetweenHeartbeats)
			}
		}
	}()

	// Periodically gather states from the server
	timer := time.NewTimer(timeBetweenPolls)
	for {
		timer.Reset(timeBetweenPolls)
		select {
		case <-timer.C: // if the timer signal is recieved, continue
		case <-a.debgForceRefresh: // continue if forced by signal
		case <-a.stop:
			return // if the stop signal is received, exit from the function
		}

		devices, err := a.getDevicesFromServer(a.context)
		if err != nil {
			a.log.Warn("error getting devices from server", "err", err)
			a.context.SendStatus(ipv4.DriverStatusError)
			return
		}

		a.log.Debug("driver got devices from server", "devices", devices)
		// Just got a batch of devices - send them to the differ to see if any devices were changed or removed
		a.differ.Consider(devices)
	}
}

// Stop stops the adapter
func (a *ipv4Adapter) Stop() { a.stop <- struct{}{} }

//
// Enacting intents
//

// EnactIntent will attempt to satisfy the provided intent by sending network
// messages to the Devices specified by target.
func (a *ipv4Adapter) EnactIntent(target types.ExternalComponentID, intent types.Intent) error {
	switch typed := intent.(type) {
	default:
		return fmt.Errorf("unhandled intent type: %T", intent)
	case types.SetLightEmitterIntent:
		return a.enactSetLightEmitterIntent(target, typed)
	}
}

func (a *ipv4Adapter) enactSetLightEmitterIntent(target types.ExternalComponentID, intent types.SetLightEmitterIntent) error {
	device, err := a.differ.GetLatest(target.Device)
	if err != nil {
		return err
	}
	component, ok := device.Components[target.Name]
	if !ok {
		return fmt.Errorf("device %v does not have a component named %v", target.Device.ID, target.Name)
	}
	_, ok = component.(types.LightEmitter)
	if !ok {
		return fmt.Errorf("cannot enact SetLightEmitterIntent on a component which is not a types.LightEmitter (got %T)", component)
	}

	// Build a component using the server-side light type
	light := Light{
		// IsOn: false,  <-- default value
		// OutputInPercent: 0,  <-- default value
	}

	if intent.BrightnessInPercent > 0 {
		light.IsOn = true
		if intent.BrightnessInPercent >= 100 {
			light.OutputInPercent = 100
		} else {
			light.OutputInPercent = intent.BrightnessInPercent
		}
	}
	return a.setComponent(a.context, target.Device.ID, target.Name, light) // Send the component to the server
}

//
// IPv4 Helper functions (retrieving data from server)
//
func (a *ipv4Adapter) getDataFromServer(context ipv4.ServiceContext) (data []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("getDataFromServer %v received panic: %v", context.IP.String(), err.Error())
		}
	}()

	var url string
	if context.Port != nil {
		url = getDevicesURL(context.IP, *context.Port)
	} else {
		url = getDevicesURL(context.IP, a.port)
	}

	var res *http.Response
	done := make(chan struct{})
	go func() {
		res, err = http.Get(url)
		done <- struct{}{}
	}()

	select {
	case <-time.After(serverCommunicationTimeout):
		err = fmt.Errorf("example ipv4 adapter timed out after waiting for response from %v for %v", serverCommunicationTimeout, url)
		return
	case <-done:
	}
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("could not read data from server (url=%s)", url)
	}
	return body, nil
}

func getStatusURL(ip net.IP, port uint16) string {
	return "http://" + ip.String() + ":" + strconv.Itoa(int(port)) + "/status"
	/*
		if context.Port == "" {
			return "http://" + context.IP.String() + ":" + strconv.Itoa(int(DefaultPort)) + "/status"
		}
		return "http://" + context.IP.String() + ":" + context.Port + "/status"
	*/
}

func getDevicesURL(ip net.IP, port uint16) string {
	return "http://" + ip.String() + ":" + strconv.Itoa(int(port)) + "/devices"
	/*
		if context.Port == "" {
			return "http://" + context.IP.String() + ":" + strconv.Itoa(int(DefaultPort)) + "/devices"
		}
		return "http://" + context.IP.String() + ":" + context.Port + "/devices"
	*/
}

func (a *ipv4Adapter) isExampleService(context ipv4.ServiceContext) bool {
	defer func() {
		if r := recover(); r != nil {
			Log.Warn("paniced during call to isExampleServer(%v): %v\n", context.IP.String(), r)
		}
	}()

	var url string
	if context.Port != nil {
		url = getStatusURL(context.IP, *context.Port)
	} else {
		url = getStatusURL(context.IP, a.port)
	}

	res, err := http.Get(url)
	if err != nil {
		a.log.Warn("service is NOT an example service, because of error during http.Get", "err", err, "url_attempted", url)
		return false
	}
	statusJSON, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		a.log.Warn("service is NOT an example service, because the content of the status path was not readable", "err", err, "url_attempted", url)
		return false
	}

	var status serverStatus
	err = json.Unmarshal(statusJSON, &status)
	if err != nil {
		a.log.Warn("service is NOT an example service, because the content of the status path was not marshalable to a status struct", "err", err, "url_attempted", url, "content_read", statusJSON)
		return false
	}

	if status.Type != serverTypeAllAtOnce {
		a.log.Warn("service is NOT an example service, because the status type was incorrect", "expected", serverTypeAllAtOnce, "got", status.Type, "url_attempted", url, "content_read", statusJSON)
		return false
	}

	return true
}

func (a *ipv4Adapter) getDevicesFromServer(context ipv4.ServiceContext) (map[types.ExternalDeviceID]types.Device, error) {
	data, err := a.getDataFromServer(context) // Get data from server
	if err != nil {
		a.log.Warn("error getting devices from server", "err", err)
		return nil, err
	}
	return parseDevices(data) // Parse data into devices
}

func parseDevices(input []byte) (map[types.ExternalDeviceID]types.Device, error) {
	// The expected format for the data is a map of keys to device definitions
	messagesByID := make(map[string]json.RawMessage)
	if err := json.Unmarshal(input, &messagesByID); err != nil {
		return nil, err
	}

	devices := make(map[types.ExternalDeviceID]types.Device)
	for id, rawMessage := range messagesByID {
		device, err := parseDevice([]byte(rawMessage))
		if err != nil {
			return nil, fmt.Errorf("error while attempting to marshal text to json: (err = %v) (text = %s)", err, rawMessage)
		}
		key := types.ExternalDeviceID{
			Manufacturer: "example",
			ID:           id,
		}
		devices[key] = device
	}
	return devices, nil
}

func parseDevice(input []byte) (types.Device, error) {
	// Cheat and use the JSON unmarshalling function already used by the server
	device, err := deviceFromJSON([]byte(input))
	if err != nil {
		return types.Device{}, err
	}
	// Convert the server device type into a sift Device
	return convertDevice(device), nil
}

// convertDevice converts a server-formatted device into a sift device
func convertDevice(d Device) types.Device {
	components := make(map[string]types.Component)
	for id, serverComponent := range d.Components {
		comp, err := convertComponent(serverComponent)
		if err != nil {
			Log.Warn(err.Error())
			continue
		}
		components[id] = comp
	}

	return types.Device{
		Components: components,
	}
}

func convertComponent(c Component) (types.Component, error) {
	switch typed := c.(type) {
	default:
		return nil, fmt.Errorf("unsupported component type %T\n", c)
	case Light:
		return convertLight(typed), nil
	}
}

// convertLight converts a server-formatted Light into a sift LightEmitter
func convertLight(light Light) types.LightEmitter {
	return types.LightEmitter{
		BaseComponent: types.BaseComponent{
			Make:  "example",
			Model: "light_emitter_1",
		},
		State: types.LightEmitterState{
			BrightnessInPercent: light.OutputInPercent,
		},
	}
}

func (a *ipv4Adapter) setComponent(context ipv4.ServiceContext, devID, compID string, comp interface{}) error {
	//typed := comp.GetTyped()           // Wrap component with its Type
	asJSON, err := json.Marshal(comp) // convert Component to JSON
	if err != nil {
		return fmt.Errorf("error converting component to JSON: %v", err)
	}

	var url string
	if context.Port != nil {
		url = getDevicesURL(context.IP, *context.Port)
	} else {
		url = getDevicesURL(context.IP, a.port)
	}
	url = url + "/" + devID + "/" + compID

	Log.Debug("sending http POST to set component", "url", url, "content", string(asJSON), "context", context)

	res, err := http.Post(url, "Application/JSON", strings.NewReader(string(asJSON))) // do POST
	if err != nil {
		return fmt.Errorf("error posting component to JSON: %v", err)
	}
	respBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading POST response body: %v", err)
	}
	res.Body.Close()

	// Here we check the status code of the response to see if the component
	// was pushed successfully. If status code alone is not enough for your
	// Adapter to determine if it was successful or not, you may need to parse
	// and interpret the response body.
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("expected response code %v from POST, got %v. Response body: %s", http.StatusOK, res.StatusCode, respBody)
	}
	return nil
}
