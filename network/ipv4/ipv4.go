package ipv4

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/upwrd/sift/logging"
	"net"
	"sync"
)

// Log is used to log messages for the example package. Logs are disabled by
// default; use sift/logging.SetLevel() to set log levels for all packages, or
// Log.SetHandler() to set a custom handler for this package (see:
// https://godoc.org/gopkg.in/inconshreveable/log15.v2)
var Log = logging.Log.New("pkg", "network/ipv4")

// AdapterStatus describes the status of an Adapter
type AdapterStatus int

var credentialsByFactoryKey = map[string]map[string]string{}

// Possible Adapter statuses
const (
	AdapterStatusIncorrectService AdapterStatus = iota // Context pointed to a service that the Adapter does not control
	AdapterStatusHandling                              // Adapter is handling the Context
	AdapterStatusDone                                  // Adapter is done handling the Context
	AdapterStatusError                                 // Adapter received an error
)

// ServiceDescription describes ipv4 characteristics of a networked service
type ServiceDescription struct {
	OpenPorts []uint16
}

// ServiceContext describes (and grants access to) a particular IPv4 service.
type ServiceContext struct {
	IP          net.IP            // the IP of the service
	Port        *uint16           // TODO REMOVE (prev: optionally specify the port of the running service)
	Credentials map[string]string //TODO REMOVE

	status      chan AdapterStatus
	slock       *sync.Mutex
	dbpath      string
	adapterName string
}

// SendStatus sends a status to the creator of the Context. Will return an
// error if the Context has been killed, otherwise nil.
func (s ServiceContext) SendStatus(ds AdapterStatus) error {
	s.slock.Lock()
	defer s.slock.Unlock()

	// If the status channel is nil, that means it has been closed by a call to
	// KillContext()
	if s.status == nil {
		return fmt.Errorf("context has been killed")
	}
	// If the status channel is non-nil, send the status
	s.status <- ds
	return nil
}

// BuildContext builds a new ServiceContext with the given IP. The second
// return value is a channel which will receive status updates from calls to
// context.SendStatus() until the Context is killed.
func BuildContext(ip net.IP, dbpath, adapterName string) (*ServiceContext, <-chan AdapterStatus) {
	status := make(chan AdapterStatus, 10)
	return &ServiceContext{
		IP:          ip,
		status:      status,
		Credentials: make(map[string]string), //TODO REMOVE
		slock:       &sync.Mutex{},
		dbpath:      dbpath,
		adapterName: adapterName,
	}, status
}

// KillContext kills the specified context. Subsequent calls on the Context
// will return errors
func KillContext(context *ServiceContext) {
	if context != nil {
		context.slock.Lock()
		defer context.slock.Unlock()
		if context.status != nil {
			close(context.status)
		}
		// Unset the status channel so future callers to SendStatus will know that
		// the Context has been killed
		context.status = nil
	}
}

// StoreData will store a string of data with the Context, which is retrievable
// with the given key.
//
// Note that Adapters with different names (e.g. "google chromecast" vs "amazon
// fire") are segregated from eachother and will not be able to read or write
// over eachother's data.
func (s ServiceContext) StoreData(key, value string) (err error) {
	if !s.isAlive() {
		return fmt.Errorf("Context is dead")
	}
	return storeData(s.dbpath, s.adapterName, key, value)
}

func storeData(dbpath, adapterName, key, value string) (err error) {
	db, err := sqlx.Connect("sqlite3", dbpath)
	if err != nil {
		return fmt.Errorf("could not establish connection to database %v: %v", dbpath, err)
	}
	defer db.Close()
	// begin a database transaction
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("could not begin transaction: %v", err)
	}
	// If something bad happens, roll back the transaction
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				Log.Error("could not roll back db transaction", "original_err", err, "rollback_err", rbErr)
			}
			Log.Warn("rolled back db transaction", "original_err", err)
		} else {
			if cmErr := tx.Commit(); cmErr != nil {
				Log.Error("could not commit transaction", "commit_err", cmErr)
				err = fmt.Errorf("could not commit transaction: %v", cmErr)
			}
			Log.Debug("adapter credentials updated; transaction committed")
		}
	}()

	// Try updating the Adapter Credentials
	q := "UPDATE adapter_credential SET value=? WHERE adapter_name=? AND key=?"
	res, err := tx.Exec(q, value, adapterName, key)
	if err != nil {
		err = fmt.Errorf("error updating adapter_credentials: %v", err)
		return
	}

	// Check the number of rows affected by the update; should be 1 if the
	// row existed, and 0 if not
	var n int64
	if n, err = res.RowsAffected(); err != nil {
		err = fmt.Errorf("error getting row count (required for update): %v", err)
		return
	} else if n == 0 {
		// The update failed, do an insert instead
		q = "INSERT INTO adapter_credential (adapter_name, key, value) VALUES (?, ?, ?)"
		_, err = tx.Exec(q, adapterName, key, value)
		if err != nil {
			err = fmt.Errorf("error inserting adapter credentials: %v", err)
			return
		}
	}
	return
}

// GetData retrieves the string data which has been stored for this context.
//
// Note that Adapters with different names (e.g. "google chromecast" vs "amazon
// fire") are segregated from eachother and will not be able to read or write
// over eachother's data.
func (s ServiceContext) GetData(key string) (string, error) {
	if !s.isAlive() {
		return "", fmt.Errorf("Context is dead")
	}
	return getData(s.dbpath, s.adapterName, key)
}

func getData(dbpath, adapterName, key string) (string, error) {
	var value string
	// Get a connection to the database
	db, err := sqlx.Connect("sqlite3", dbpath)
	if err != nil {
		return "", fmt.Errorf("could not establish connection to database %v: %v", dbpath, err)
	}
	defer db.Close()

	// Get the credentials
	q := "SELECT value FROM adapter_credential WHERE adapter_name=? AND key=?"
	if err := db.Get(&value, q, adapterName, key); err != nil {
		return "", fmt.Errorf("could not get credentials from database: %v", err)
	}
	return value, nil
}

func (s ServiceContext) isAlive() bool {
	s.slock.Lock()
	defer s.slock.Unlock()
	return s.status != nil
}
