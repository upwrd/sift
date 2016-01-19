package example

// server.go defines an example backend server.
// It's format is intentionally different than the SIFT format, so the driver
// has to do some translation. Use it as an example when thinking about how you
// can implement your driver. :)

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"
	"gopkg.in/tylerb/graceful.v1"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"
)

const (
	serverTypeAllAtOnce = "all_at_once"
	defaultPort         = uint16(10203)
	testPort            = uint16(1)
)

const (
	componentTypeLight = "light"
	componentTypeLock  = "lock"
)

// A Device is an example server's representation of a physical unit
type Device struct {
	Components map[string]Component
}

// A Component is an example server's representation of a functional unit
type Component interface {
	GetType() string
}

// A Light is an example server's representation of a light
type Light struct {
	IsOn            bool  `json:"is_on,omitempty"`
	OutputInPercent uint8 `json:"output_in_percent,omitempty"`
}

// GetType returns the light's type
func (l Light) GetType() string { return componentTypeLight }

// MarshalJSON uses the 'typed' version of the Light when marshalling to JSON
func (l Light) MarshalJSON() ([]byte, error) {
	s := struct {
		IsOn            bool  `json:"is_on,omitempty"`
		OutputInPercent uint8 `json:"output_in_percent,omitempty"`
		Type            string
	}{
		IsOn:            l.IsOn,
		OutputInPercent: l.OutputInPercent,
		Type:            componentTypeLight,
	}
	return json.Marshal(s)
}

// A Lock is an example server's representation of a lock
type Lock struct {
	IsOpen bool `json:"is_open"`
}

// GetType returns the lock's type
func (l Lock) GetType() string { return componentTypeLock }

// MarshalJSON uses the 'typed' version of the Lock when marshalling to JSON
func (l Lock) MarshalJSON() ([]byte, error) {
	s := struct {
		IsOpen bool `json:"is_open"`
		Type   string
	}{
		IsOpen: l.IsOpen,
		Type:   componentTypeLock,
	}
	return json.Marshal(s)
}

type serverConfig struct {
	version           string
	pushEnabled       bool // enable push notifications of Device/Component changes
	componentIndexing bool // enable GET/POST to specific Components
}

// A Server runs the example service. It mimics a hub-like service that
// maintains a collection of different Devices. The server's data model is
// intentionally different from the standard SIFT data types to demonstrate
// how one might translate real-world services.
type Server struct {
	port        uint16
	devices     map[string]Device
	notify      chan struct{}
	listeners   []chan bool
	netListener net.Listener
	httpServer  *graceful.Server
	log         log.Logger
	stop        chan struct{}
}

// NewServer properly instantiates a Server
func NewServer(port uint16) *Server {
	return &Server{
		port:      uint16(port),
		devices:   make(map[string]Device),
		notify:    make(chan struct{}, 10),
		listeners: make([]chan bool, 0),
		log:       Log.New("obj", "example_server", "id", logext.RandId(8)),
		stop:      make(chan struct{}),
	}
}

// Serve begins running the server. If provided with a valid port, the Server
// will serve the example service over a local RESTful http API.
func (s *Server) Serve() {
	if s.port != testPort { // in test mode, the server does not serve over HTTP
		go s.serveHTTP()
	}

	for {
		select {
		case <-s.stop:
			closeChan := s.stopHTTP()
			select {
			case _, isClosed := <-closeChan:
				if !isClosed {
					s.log.Crit("something went wrong!")
				}
			case <-time.After(10 * time.Second):
				s.log.Warn("taking longer than 10 seconds to close http server, skipping")
			}
		case <-s.notify:
			for _, listener := range s.listeners {
				listener <- true
			}
		}
	}
}

// Stop stops the Server
func (s *Server) Stop() {
	s.stop <- struct{}{}
}

func (s *Server) serveHTTP() {
	addr := ":" + strconv.Itoa(int(s.port)) // ex ":10123"
	s.httpServer = &graceful.Server{
		Timeout: 2 * time.Second,

		Server: &http.Server{
			Addr:    addr,
			Handler: s.Handlers(fmt.Sprintf("http://localhost:%v", s.port)),
		},
	}
	s.log.Info("example http server started", "root", fmt.Sprintf("http://localhost:%v/", s.port))
	s.log.Error("example server error while serving", "err", s.httpServer.ListenAndServe())
}

func (s *Server) stopHTTP() <-chan struct{} {
	s.httpServer.Stop(5 * time.Second)
	return s.httpServer.StopChan()
}

func (s *Server) close() error {
	if s.netListener != nil {
		return s.netListener.Close()
	}
	return nil
}

// SetDevice sets the server to report the given device with id "id"
func (s *Server) SetDevice(id string, device Device) {
	s.devices[id] = device
	s.notify <- struct{}{}
}

func (s *Server) setComponent(deviceID, componentID string, c Component) error {
	device, ok := s.devices[deviceID]
	if !ok {
		return fmt.Errorf("no device found with id %v", deviceID)
	}

	device.Components[componentID] = c
	s.notify <- struct{}{}
	return nil
}

func (s *Server) removeDevice(id string) {
	delete(s.devices, id)
	s.notify <- struct{}{}
}

func (s *Server) listenForChanges(hasChanged chan bool) {
	s.listeners = append(s.listeners, hasChanged)
}

// Handlers provides the set of HTTP route handlers which make up the example
// service API.
func (s *Server) Handlers(prefix string) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", dumpRoutesFn(prefix)).Methods("GET")
	r.HandleFunc("/status", statusHTTP).Methods("GET")
	r.HandleFunc("/devices", s.getDevicesHTTP).Methods("GET")
	r.HandleFunc("/devices/{devID}", s.getDeviceHTTP).Methods("GET")
	r.HandleFunc("/devices/{devID}", s.setDeviceHTTP).Methods("POST")
	r.HandleFunc("/devices/{devID}/{compID}", s.getComponentHTTP).Methods("GET")
	r.HandleFunc("/devices/{devID}/{compID}", s.setComponentHTTP).Methods("POST")
	return r
}

var hardcodedPaths = []string{"/", "/status", "/devices", "/devices/{devID}", "/devices/{devID}/{compID}"}

//
func (s *Server) getDevices() map[string]Device {
	return s.devices
}

func (s *Server) getDevicesHTTP(w http.ResponseWriter, r *http.Request) {
	asJSON, err := json.Marshal(s.devices)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writePretty(w, asJSON, http.StatusOK)
}

func (s *Server) getDevice(id string) (Device, bool) {
	dev, ok := s.devices[id]
	return dev, ok
}

func (s *Server) getDeviceHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["devID"]
	if id == "" {
		http.Error(w, "must provide device ID", http.StatusBadRequest)
		return
	}

	device, ok := s.devices[id]
	if !ok {
		http.Error(w, "device "+id+" not found", http.StatusNotFound)
		return
	}

	asJSON, err := json.Marshal(device)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writePretty(w, asJSON, http.StatusOK)
}

func (s *Server) setDeviceHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["devID"]
	if id == "" {
		http.Error(w, "must provide device ID", http.StatusBadRequest)
		return
	}

	// Get the body of the request, which should contain a json-encoded Device
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "error reading request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Parse the request as a Device
	asDevice, err := deviceFromJSON(body)
	if err != nil {
		http.Error(w, "unable to interpret request body as valid Device: "+err.Error(), http.StatusBadRequest)
		return
	}

	s.SetDevice(id, asDevice)
	http.Redirect(w, r, "/devices/"+id, http.StatusSeeOther)
}

func (s *Server) getComponentHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	devID := vars["devID"]
	compID := vars["compID"]
	if devID == "" || compID == "" {
		http.Error(w, "must provide device ID and component ID", http.StatusBadRequest)
		return
	}

	device, ok := s.devices[devID]
	if !ok {
		http.Error(w, "device "+devID+" not found", http.StatusNotFound)
		return
	}
	comp, ok := device.Components[compID]
	if !ok {
		http.Error(w, "component "+compID+" not found on device "+devID, http.StatusNotFound)
		return
	}

	asJSON, err := json.Marshal(comp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writePretty(w, asJSON, http.StatusOK)
}

func (s *Server) setComponentHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	devID := vars["devID"]
	compID := vars["compID"]
	if devID == "" || compID == "" {
		http.Error(w, "must provide device ID and component ID", http.StatusBadRequest)
		return
	}

	// Get the body of the request, which should contain a json-encoded Component
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "error reading request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Parse the request as a Component
	asComponent, err := componentFromJSON(body)
	if err != nil {
		http.Error(w, "unable to interpret request body as valid Component: "+err.Error(), http.StatusBadRequest)
		return
	}

	s.setComponent(devID, compID, asComponent)
	http.Redirect(w, r, "/devices/"+devID+"/"+compID, http.StatusSeeOther)
}

// WritePretty tries to pretty-format a JSON string and write it to the provided http.ResponseWriter.
// If pretty-formatting fails, WritePretty will write the original string instead.
func writePretty(rw http.ResponseWriter, jsn []byte, httpCode int) {
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	prettyJSON, err := prettyFormatJSON(jsn)
	if err != nil {
		http.Error(rw, string(jsn), httpCode)
		return
	}
	http.Error(rw, string(prettyJSON), httpCode)
}

func prettyFormatJSON(input []byte) ([]byte, error) {
	var output bytes.Buffer
	err := json.Indent(&output, input, "", "   ")
	return output.Bytes(), err
}

type serverStatus struct {
	Type string `json:"type"`
}

func statusHTTP(w http.ResponseWriter, r *http.Request) {
	response := serverStatus{
		Type: serverTypeAllAtOnce,
	}

	asJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writePretty(w, asJSON, http.StatusOK)
}

func dumpRoutesFn(prefix string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		routes := getRouteURLs(prefix, hardcodedPaths)
		asJSON, err := json.Marshal(routes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writePretty(w, asJSON, http.StatusOK)
	}
}

func getRouteURLs(prefix string, paths []string) []string {
	routeURLs := make([]string, len(paths))
	for i, path := range paths {
		routeURLs[i] = prefix + path
	}
	return routeURLs
}

type rawDevice struct {
	Components map[string]json.RawMessage
}

func devicesFromJSON(input []byte) (map[string]Device, error) {
	var deviceJSONs map[string]json.RawMessage
	if err := json.Unmarshal(input, &deviceJSONs); err != nil {
		return nil, err
	}

	devices := make(map[string]Device)
	for id, deviceJSON := range deviceJSONs {
		device, err := deviceFromJSON([]byte(deviceJSON))
		if err != nil {
			return nil, err
		}
		devices[id] = device
	}
	return devices, nil
}

func deviceFromJSON(input []byte) (Device, error) {
	var rawDev rawDevice
	if err := json.Unmarshal(input, &rawDev); err != nil {
		return Device{}, err
	}

	components := make(map[string]Component)
	for id, rawComponent := range rawDev.Components {
		component, err := componentFromJSON([]byte(rawComponent))
		if err != nil {
			return Device{}, err
		}
		components[id] = component
	}

	return Device{
		Components: components,
	}, nil
}

func componentFromJSON(input []byte) (Component, error) {
	// Unmarshal as baseComponent to get Type
	base := struct {
		Type string
	}{}
	if err := json.Unmarshal(input, &base); err != nil {
		return nil, err
	}

	switch base.Type {
	default:
		return nil, fmt.Errorf("unknown type %s", base.Type)
	case componentTypeLight:
		return lightFromJSON(input)
	case componentTypeLock:
		return lockFromJSON(input)
	}
}

func lightFromJSON(input []byte) (Light, error) {
	var light Light
	err := json.Unmarshal(input, &light)
	return light, err
}

func lockFromJSON(input []byte) (Lock, error) {
	var lock Lock
	err := json.Unmarshal(input, &lock)
	return lock, err
}
