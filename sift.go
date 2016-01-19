// Package sift - the Simple Interface of Functional Things.
//
// SIFT makes it easy for developers to write code which understands and
// manipulates connected devices. A SIFT Server presents an authoritative,
// centralized repository of Devices (physical units) and their functional
// Components. SIFT Components are generically typed, meaning a developer can
// manipulate any number of Light Emitters or Media Players without
// understanding their specific implementations (e.g. Philips Hue / Google
// Chromecast) and their implementation details (such as wireless protocols
// and API specifics).
package sift

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/thejerf/suture"
	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"
	"os"
	"os/signal"
	"sift/auth"
	"sift/db"
	"sift/drivers"
	"sift/drivers/chromecast"
	cbtcp "sift/drivers/connectedbytcp"
	"sift/drivers/example"
	"sift/lib"
	"sift/logging"
	"sift/network/ipv4"
	"sift/notif"
	"sift/types"
	"sync"
	"syscall"
	"time"
)

// A suggested default database name.
const DefaultDBFilepath = "sift.db"

const (
	ipv4ScanFrequency           = 5 * time.Second
	adapterTimeout              = 15 * time.Second
	updateChanWidth             = 1000
	numAdapterUpdateListeners   = 5
	numConfirmedUpdateListeners = 5
)

// Log is used to log messages for the sift package. Logs are disabled by
// default; use sift/logging.SetLevel() to set log levels for all packages, or
// Log.SetHandler() to set a custom handler for this package (see:
// https://godoc.org/gopkg.in/inconshreveable/log15.v2)
var Log = logging.Log.New("pkg", "sift")

// SetLogLevel is a convenient wrapper to sift/logging.SetLevelStr()
// Use it to quickly set the log level for all loggers in this package.
// For more control over logging, manipulate logging.Log directly, as outlined
// by https://godoc.org/gopkg.in/inconshreveable/log15.v2#hdr-Library_Use
// (see also: github.com/inconshreveable/log15)
func SetLogLevel(lvlstr string) { logging.SetLevelStr(lvlstr) }

// Server maintains the state of SIFT objects and provides methods for
// listening to, retrieving, and manipulating them.
// You should always initialize Servers with a call to NewServer()
type Server struct {
	// SiftDB provides direct access to the underlying sqlite database through
	// Jason Moiron's wonderful sqlx API (see: github.com/jmoiron/sqlx)
	*db.SiftDB
	dbpath string

	auth.Authorizor // Provides login/authorize methods
	notif.Provider  // Provides notification pub/sub methods
	notif.Receiver  // Adds methods to post notifications

	// Factories, adapters and their updates
	factoriesByDescriptionID map[string]drivers.AdapterFactory
	adapters                 map[string]drivers.Adapter
	updatesFromAdapters      chan updatePackage
	prioritizer              lib.IPrioritizer

	// Scanners
	ipv4Scan ipv4.IContinuousScanner

	// Others
	stop                    chan struct{}
	stopped                 chan struct{}
	interceptingStopSignals bool
	log                     log.Logger
}

// NewServer constructs a new SIFT Server, using the SIFT database at the
// provided path (or creating a new one it does not exist). Be sure to start
// the Server with Serve()
func NewServer(dbpath string) (*Server, error) {
	newDB, err := db.Open(dbpath)
	if err != nil {
		return nil, fmt.Errorf("could not open sift db: %v", err)
	}
	authorizor := auth.New()
	notifier := notif.New(authorizor)

	return &Server{
		SiftDB: newDB,
		dbpath: dbpath,

		Authorizor: authorizor,
		Provider:   notifier,
		Receiver:   notifier,

		factoriesByDescriptionID: make(map[string]drivers.AdapterFactory),
		adapters:                 make(map[string]drivers.Adapter),
		updatesFromAdapters:      make(chan updatePackage, updateChanWidth),
		prioritizer:              lib.NewPrioritizer(nil), // uses default sorting

		ipv4Scan: ipv4.NewContinousScanner(ipv4ScanFrequency),

		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
		log:     Log.New("obj", "server", "id", logext.RandId(8)),
	}, nil
}

// Serve starts running the SIFT server. Most of the time you'll want to call
// in a goroutine; or as a Suture Service (see github.com/thejerf/suture)
func (s *Server) Serve() {
	s.stopOnExitSignal() // capture ^c and SIGTERM, and close gracefully (see: http://stackoverflow.com/a/18158859/3088592)
	supervisor := suture.NewSimple("sift server")
	supervisor.Add(s.ipv4Scan)
	go supervisor.ServeBackground()

	// Listen for updates from adapters and consider them.
	var wg1 sync.WaitGroup
	for i := 0; i < numAdapterUpdateListeners; i++ {
		wg1.Add(1)
		go func() {
			for update := range s.updatesFromAdapters {
				// Consider the update through the prioritizer
				if err := s.prioritizer.Consider(update.AdapterDescription, update.update); err != nil {
					s.log.Error("error while prioritizing update", "err", err)
				}
			}
			wg1.Done()
		}()
	}

	// Listen for updates from the prioritizer channel. These are updates
	// which should be reflected in the sift data model.
	var wg2 sync.WaitGroup
	for i := 0; i < numConfirmedUpdateListeners; i++ {
		wg2.Add(1)
		go func() {
			for update := range s.prioritizer.OutputChan() {
				s.handleUpdate(update)
			}
			wg2.Done()
		}()
	}

	// Wait for less-frequent signals
	for {
		select {
		case <-s.stop:
			// stop gracefully
			s.log.Debug("sift server stopping due to stop signal")
			//			stopTimer := time.NewTimer(30 * time.Second)
			//			workersStopped := make(chan struct{})
			//			go func() {
			//				close(s.updatesFromAdapters) // will cause adapter update listeners to stop
			//				wg1.Wait()
			//				// TODO: CLOSE PRIORITIZER
			//				wg2.Wait()
			//				workersStopped <- struct{}{}
			//			}()
			//			select {
			//			case <-stopTimer.C:
			//				s.log.Warn("timed out waiting for workers to stop, closing instead")
			//			case <-workersStopped:
			//			}
			if err := s.SiftDB.Close(); err != nil {
				s.log.Crit("could not gracefully close sift database", "err", err)
			}
			s.stopped <- struct{}{}
			return
		case ipv4Service := <-s.ipv4Scan.FoundServices():
			// new IPv4 service found
			go s.tryHandlingIPv4Service(ipv4Service)
		}
	}
}

// Stop stops the Server (does not block)
func (s *Server) Stop() {
	s.stop <- struct{}{}
}

// StopAndWait stops the SIFT server and waits for it to complete.
func (s *Server) StopAndWait(timeout time.Duration) error {
	s.Stop()
	select {
	case <-s.stopped:
		return nil
	case <-time.NewTimer(timeout).C:
		return fmt.Errorf("timed out after %v", timeout)
	}
}

type updatePackage struct {
	lib.AdapterDescription
	update interface{}
}

// AddAdapterFactory adds an AdapterFactory to the Server. Once added, the
// Server will begin searching for services matching the AdapterFactory's
// description. If any are found, the Server will use the AdapterFactory to
// create an Adapter to handle the sevice.
func (s *Server) AddAdapterFactory(factory drivers.AdapterFactory) (string, error) {
	var id string
	switch typed := factory.(type) {
	default:
		s.log.Warn("unhandled adapter factory type", "name", factory.Name(), "type", fmt.Sprintf("%T", factory))
		return "", fmt.Errorf("unhandled adapter factory type: %T", factory)
	case drivers.IPv4DriverFactory:
		id = s.ipv4Scan.AddDescription(typed.GetIPv4Description())
		s.factoriesByDescriptionID[id] = factory
	}
	s.log.Info("added adapter factory", "name", factory.Name(), "id", id)
	return id, nil
}

var defaultAdapterFactories = []func() drivers.AdapterFactory{
	func() drivers.AdapterFactory { return example.NewFactory(55442) }, // SIFT example server
	func() drivers.AdapterFactory { return cbtcp.NewFactory() },        // Connected by TCP
	func() drivers.AdapterFactory { return chromecast.NewFactory() },   // Google Chromecast
}

// AddDefaults adds default Adapter Factories to the SIFT server
func (s *Server) AddDefaults() error {
	for _, factoryFn := range defaultAdapterFactories {
		factory := factoryFn()
		if _, err := s.AddAdapterFactory(factory); err != nil {
			return fmt.Errorf("error adding default factory %+v: %v", factory, err)
		}
	}
	return nil
}

func (s *Server) addAdapter(adapter drivers.Adapter) string {
	id := uuid.New()
	s.adapters[id] = adapter
	return id
}

func (s *Server) removeAdapter(id string) {
	delete(s.adapters, id)
}

//
// Handling updates from adapters
//
func (s *Server) handleUpdate(update interface{}) {
	switch typed := update.(type) {
	case lib.DeviceUpdated:
		s.log.Debug("received notice of updated device", "update", typed)
		s.handleDeviceUpdated(typed)
	case lib.DeviceDeleted:
		s.log.Debug("received notice of deleted device", "delete", typed)
		s.handleDeviceDeleted(typed)
	}
}

func (s *Server) handleDeviceUpdated(update lib.DeviceUpdated) {
	s.log.Debug("handling device update", "update", update)
	// upsert the updated Device, and get the changes
	resp, err := s.SiftDB.UpsertDevice(update.ID, update.NewState)
	if err != nil {
		s.log.Warn("could not upsert device indicated in update", "err", err, "update", update)
		panic(fmt.Sprintf("could not upsert device indicated in update: %v", err))
	}

	// notify listeners of changes
	for name, comp := range resp.UpsertedComponents {
		id := types.ComponentID{Name: name, DeviceID: resp.DeviceID}
		s.PostComponent(id, comp, notif.Update)
	}
	for name, comp := range resp.DeletedComponents {
		id := types.ComponentID{Name: name, DeviceID: resp.DeviceID}
		s.PostComponent(id, comp, notif.Delete)
	}
	if resp.HasDeviceChanged {
		s.PostDevice(resp.DeviceID, update.NewState, notif.Update)
	}
}

func (s *Server) handleDeviceDeleted(update lib.DeviceDeleted) {
	s.log.Crit("STUB: server2.handleDeviceDeleted()", "update", update)
}

// EnactIntent attempts to fulfill an intent, usually to change the state of
// a particular Component. For a list of possible intents, see sift/types
func (s *Server) EnactIntent(target types.ComponentID, intent types.Intent) error {
	if err := s.sanityCheck(); err != nil {
		return err
	}
	s.log.Debug("submitting intent", "target", target, "intent", intent)

	// Translate the internal intent (using types.ComponentID) into an external
	// intent (using types.ExternalComponentID)
	externalDevID, err := s.SiftDB.GetExternalDeviceID(target.DeviceID)
	if err != nil {
		return fmt.Errorf("could not get external ID for device %v: %v", target.DeviceID, err)
	}
	if externalDevID.Manufacturer == "" || externalDevID.ID == "" {
		return fmt.Errorf("no active adapters are currently handling component %v", target)
	}

	// Determine the highest-priority Adapter currently serving the connected Device
	adapterID := s.prioritizer.GetHighestPriorityAdapterForDevice(externalDevID)
	adapter, ok := s.adapters[adapterID]
	if !ok {
		return fmt.Errorf("could not find adapter matching highest priority ID '%v': %v", adapterID, err)
	}

	// Pass the external intent to the Adapter
	externalTarget := types.ExternalComponentID{Device: externalDevID, Name: target.Name}
	s.log.Debug("passing external intent to adapter", "target", target, "intent", intent, "adapter", adapter)
	return adapter.EnactIntent(externalTarget, intent)
}

//IPv4

// tryHandlingIPv4Service will walk through each of the provided
func (s *Server) tryHandlingIPv4Service(n ipv4.ServiceFoundNotification) {
	for _, id := range n.MatchingDescriptionIDs {
		// Find the factory matching the id
		if factory, ok := s.factoriesByDescriptionID[id]; ok {
			// ...it should be an IPv4 Factory
			if asIPv4Factory, ok := factory.(drivers.IPv4DriverFactory); !ok {
				s.log.Error("expected an IPv4 factory, got something different!", "got", fmt.Sprintf("%T", factory))
			} else {
				// build a context for the given IP
				context, statusChan := ipv4.BuildContext(n.IP, s.dbpath, factory.Name())

				// build a new adapter from the factory, which will attempt to handle the context
				adapter := asIPv4Factory.HandleIPv4(context)
				adapterID := s.addAdapter(adapter)

				// keep listening to the adapter until it fails or times out
				adapterDied := make(chan bool, 10)
				go func() {
					for update := range adapter.UpdateChan() {
						// pass the update and adapter to the main channel
						pkg := updatePackage{
							AdapterDescription: lib.AdapterDescription{
								Type: lib.ControllerTypeIPv4,
								ID:   adapterID,
							},
							update: update,
						}
						// This passes the update to be eval'd by the server
						s.updatesFromAdapters <- pkg
						// This signals that the Adapter should be allowed to
						// continue handling the context (NOTE: it should also be
						// heartbeating)
						adapterDied <- false
					}
					adapterDied <- true
				}()

				timer := time.NewTimer(adapterTimeout)
			waitUntilFailure:
				for {
					timer.Reset(adapterTimeout)

					select {
					case <-timer.C:
						s.log.Debug("adapter timed out")

						break waitUntilFailure // timed out
					case itDied := <-adapterDied:
						// if it died, break. if it didn't, keep going
						if itDied {
							s.log.Debug("adapter died")
							break waitUntilFailure
						}
					case status, more := <-statusChan:
						if !more {
							s.log.Debug("adapter status channel closed")
							break waitUntilFailure
						}
						// The adapter can send messages through the context.Status channel.
						// If the value is ipv4.DriverStatusHandling, it is treated as a
						// keep-alive heartbeat message. Any other status indicates that
						// the adapter is no longer handling the service.
						if status != ipv4.DriverStatusHandling {
							s.log.Debug("adapter returned non-handling status", "status", status)
							break waitUntilFailure
						}
					}
				}

				// If we've reached this point, the adapter is done.
				// Kill it, and move on to the next viable adapter
				ipv4.KillContext(context)
				s.removeAdapter(adapterID)
			}
		}
	}
	// If we've reached this point, all viable adapters (if any) have failed.
	// Release the IP; if it's still there, it will be picked up on the next
	// scan and tried again.
	s.ipv4Scan.Unlock(n.IP)
}

func (s *Server) sanityCheck() error {
	if s == nil {
		return fmt.Errorf("Server receiver cannot be nil")
	}

	if s.prioritizer == nil {
		st, err := NewServer(DefaultDBFilepath)
		if err != nil {
			return fmt.Errorf("error instantiating SIFT server: %v", err)
		}
		s = st
	}
	return nil
}

func (s *Server) stopOnExitSignal() {
	if !s.interceptingStopSignals {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		signal.Notify(c, syscall.SIGTERM)
		go func() {
			<-c
			fmt.Printf("\n\ncaught SIGTERM, shutting down gracefully...")
			s.Stop()
			<-s.stopped
			os.Exit(1)
		}()
	}
}
