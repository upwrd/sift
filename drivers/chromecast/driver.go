package chromecast

import (
	"bufio"
	"encoding/json"
	"fmt"
	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"
	"os/exec"
	"github.com/upwrd/sift/lib"
	"github.com/upwrd/sift/logging"
	"github.com/upwrd/sift/network/ipv4"
	"github.com/upwrd/sift/types"
	"github.com/upwrd/sift/drivers"
	"time"
)

// Log is used to log messages for the chromecast package. Logs are disabled
// by default; use sift/logging.SetLevel() to set log levels for all packages,
// or Log.SetHandler() to set a custom handler for this package (see:
// https://godoc.org/gopkg.in/inconshreveable/log15.v2)
var Log = logging.Log.New("pkg", "drivers/chromecast")

const (
	manufacturer          = "google"
	timeBetweenHeartbeats = 5 * time.Second
)

var openPorts = []uint16{8008, 8009}

// An AdapterFactory creates adapters
type AdapterFactory struct{}

// NewFactory properly instantiates a new AdapterFactory
func NewFactory() *AdapterFactory {
	return &AdapterFactory{}
}

// HandleIPv4 spawns a new Adapter to handle an IPv4 context
func (f *AdapterFactory) HandleIPv4(context ipv4.ServiceContext) drivers.Adapter {
	return newAdapter(context)
}

// GetIPv4Description returns a description of the example IPv4 service that
// can be used to identify example services on a network
func (f *AdapterFactory) GetIPv4Description() ipv4.ServiceDescription {
	return ipv4.ServiceDescription{OpenPorts: openPorts}
}

// Name returns the name of this adapter factory, "Google Chromecast"
func (f *AdapterFactory) Name() string { return "Google Chromecast" }

type ipv4Adapter struct {
	updateChan chan interface{}
	context    ipv4.ServiceContext
	differ     lib.SetOutputBasedDeviceDiffer
	desc       lib.AdapterDescription
	stop       chan struct{}
	log        log.Logger
}

func newAdapter(context ipv4.ServiceContext) *ipv4Adapter {
	log := Log.New("obj", "Google Chromecast ipv4 adapter", "id", logext.RandId(8), "adapting", context.IP.String())
	log.Info("Google Chromecast adapter created", "adapting", context.IP.String())
	adapter := &ipv4Adapter{
		updateChan: make(chan interface{}, 100),
		context:    context,
		differ:     lib.NewAllAtOnceDiffer(),
		stop:       make(chan struct{}),
		log:        log,
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

type pyMsg struct {
	Type string
}

type pyError struct {
	pyMsg
	Error string
}

type pyUpdate struct {
	pyMsg
	Update types.MediaPlayer
}

// Serve begins adapting the example service specified by the adapter's
// context. As updates within the service are found, they will be sent to the
// update channel provided by UpdateChan(). While the adapter is serving,
// heartbeat messages will be sent to the adapter's context's status channel.
func (a *ipv4Adapter) Serve() {
	a.log.Debug("serving")

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

	// Start the python script which will monitor this Chromecast service
	cmd := exec.Command("python", "../drivers/chromecast/get_updates.py", a.context.IP.String())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("err cmt.StdoutPipe(): %v\n", err)
	}
	if err := cmd.Start(); err != nil {
		fmt.Printf("err cmd.Start(): %v\n", err)
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		msg := scanner.Text()
		asPyMsg := pyMsg{}
		if err = json.Unmarshal([]byte(msg), &asPyMsg); err != nil {
			a.log.Error("could not unmarshal message from python script", "msg", msg, "err", err)
			return
		}

		switch asPyMsg.Type {
		default:
			a.log.Error("got invalid message type from python script (or could not parse appropriately)", "msg", msg, "msg_type", asPyMsg.Type)
		case "error":
			asPyError := pyError{}
			if err = json.Unmarshal([]byte(msg), &asPyError); err != nil {
				a.log.Error("could not unmarshal error message from python script", "msg", msg, "err", err)
				return
			}
			a.log.Error("error from python script", "err", asPyError.Error)
		case "update":
			asPyUpdate := pyUpdate{}
			if err = json.Unmarshal([]byte(msg), &asPyUpdate); err != nil {
				a.log.Error("could not unmarshal update message from python script", "msg", msg, "err", err)
				return
			}

			a.log.Debug("update from python script", "update", asPyUpdate.Update, "msg", msg)
			newDevice := types.Device{
				Name:     fmt.Sprintf("Chromecast @ %s", a.context.IP.String()),
				IsOnline: true,
				Components: map[string]types.Component{
					"chromecast": asPyUpdate.Update,
				},
			}
			newDeviceExternal := types.ExternalDeviceID{
				Manufacturer: manufacturer, // google
				ID:           "Chromecast @ " + a.context.IP.String(),
			}
			newDevices := map[types.ExternalDeviceID]types.Device{newDeviceExternal: newDevice}
			a.differ.Consider(newDevices)
		}
	}
	if err := scanner.Err(); err != nil {
		a.log.Error("could not scan from python output scanner", "err", err)
		return
	}
	if err := cmd.Wait(); err != nil {
		a.log.Error("error while waiting for python script to end", "err", err)
		return
	}
}

// Stop stops the adapter
func (a *ipv4Adapter) Stop() { a.stop <- struct{}{} }

// EnactIntent will attempt to satisfy the provided intent by sending network
// messages to the Devices specified by target.
func (a *ipv4Adapter) EnactIntent(target types.ExternalComponentID, intent types.Intent) error {
	switch intent.(type) {
	default:
		return fmt.Errorf("unhandled intent type: %T", intent)
	}
}
