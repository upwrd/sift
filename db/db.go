package db

import (
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	rawsql "github.com/upwrd/sift/db/sql"
	"github.com/upwrd/sift/lib"
	"github.com/upwrd/sift/logging"
	"github.com/upwrd/sift/types"
	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"
	"io/ioutil"
	"os"

	// imports sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

// Log is used to log messages for the auth package. Logs are disabled by
// default; use sift/logging.SetLevel() to set log levels for all packages, or
// Log.SetHandler() to set a custom handler for this package (see:
// https://godoc.org/gopkg.in/inconshreveable/log15.v2)
var Log = logging.Log.New("pkg", "db/components")

// ExpansionFlags are used when retrieving SIFT Devices and Components to
// specify which parts should be gathered (expanded) from the database
type ExpansionFlags byte

// Expansion flags
const (
	ExpandNone = 0
	ExpandAll  = 1 << iota
	ExpandSpecs
	ExpandStats
)

// Device contains fields matching those in the 'device' table in the SIFT
// database, which is useful when querying the database using sqlx.
type Device struct {
	ID           int64
	Manufacturer string
	ExternalID   string `db:"external_id"`
	Name         sql.NullString
	LocationID   sql.NullInt64 `db:"location_id"`
	IsOnline     bool          `db:"is_online"`
}

// Component contains fields matching those in the 'component' table in the
// SIFT database, which is useful when querying the database using sqlx.
type Component struct {
	ID                      int64
	DeviceID                int64 `db:"device_id"`
	Name, Make, Model, Type string
}

// Location contains fields matching those in the 'location' table in the SIFT
// database, which is useful when querying the database using sqlx.
type Location struct {
	ID   int64
	Name string
}

// A SiftDB manages interactions with the underlying SIFT database
type SiftDB struct {
	//*sqlx.DB
	dbpath   string
	tempFile *os.File
	log      log.Logger
}

// Open opens the SIFT database at the provided file path. If the file does not
// exist, it is created and initialized with the SIFT schema.
func Open(pathToDBFile string) (*SiftDB, error) {
	var tempFile *os.File // will be populated if caller requests a temporary file
	switch pathToDBFile {
	case "":
		file, err := ioutil.TempFile(os.TempDir(), "siftdb_")
		if err != nil {
			return nil, fmt.Errorf("could not open temporary DB file: %v", err)
		}
		tempFile = file
		pathToDBFile = file.Name()
	case ":memory:":
		return nil, fmt.Errorf("SiftDB cannot be opened with :memory:")
	}

	// Open a connection to the file at the specified path
	db, err := sqlx.Connect("sqlite3", pathToDBFile)
	if err != nil {
		Log.Error("could not open database", "err", err, "filename", pathToDBFile)
		return nil, fmt.Errorf("could not open database at path %v: %v", pathToDBFile, err)
	}
	defer db.Close()

	// Check that this is a valid SIFT DB. If it isn't, initialize it.
	if validErr := isDBValid(db); validErr != nil {
		Log.Debug("could not validate database; this may be a new file. Initializing", "filename", pathToDBFile, "validation_error", validErr)
		if err := dbInitByGoFile(db); err != nil {
			return nil, fmt.Errorf("error initializing sift DB: %v", err)
		}
	}

	return &SiftDB{
		dbpath:   pathToDBFile,
		tempFile: tempFile,
		log:      Log.New("obj", "components_db", "id", logext.RandId(8)),
	}, nil
}

// DB returns a connection to the sift database
func (sdb *SiftDB) DB() (*sqlx.DB, error) {
	return sqlx.Connect("sqlite3", sdb.dbpath)
}

// Close closes any temporary files that were used
func (sdb *SiftDB) Close() error {
	// Mark all Devices as inactive
	if err := sdb.markAllDevicesInactive(); err != nil {
		sdb.log.Crit("could not mark devices inactive as SIFT DB is closed")
		return err
	}

	if sdb.tempFile != nil {
		return sdb.tempFile.Close()
	}
	return nil
}

var numColumnsByTable = map[string]int{
	"component":          6,
	"device":             6,
	"light_emitter_spec": 5,
}

// isDBValid checks if the given db is a SIFT DB
func isDBValid(db *sqlx.DB) error {
	for table, numColumns := range numColumnsByTable {
		q := "PRAGMA table_info(" + table + ")"
		//rows, err := db.Queryx("PRAGMA table_info(?)", table)
		rows, err := db.Queryx(q)
		if err != nil {
			return fmt.Errorf("error trying to find number of columns in table %v: %v", table, err)
		}
		count := 0
		for rows.Next() {
			count++
		}
		if count != numColumns {
			return fmt.Errorf("number of columns on database (%v) != num expected (%v)", count, numColumns)
		}
	}
	return nil
}

// dbInit initializes the database
func dbInit(db *sqlx.DB) error {
	sqlPaths := []string{"sql2/init.sql", "sql2/populate_specs.sql"}
	for _, path := range sqlPaths {
		// Load the init SQL file
		initSQL, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("could not read %v: %v", path, err)
		}
		// Exec it
		_, err = db.Exec(string(initSQL))
		if err != nil {
			return fmt.Errorf("error while execing %v: %v", path, err)
		}
	}
	return nil
}

// TODO: implement import from .sql files which is portable.
// May require embedding files in the binary; see https://github.com/jteeuwen/go-bindata
func dbInitByGoFile(db *sqlx.DB) error {
	sqls := []string{rawsql.InitSql, rawsql.PopulateSpecsSQL}
	for i, sql := range sqls {
		// Exec it
		_, err := db.Exec(string(sql))
		if err != nil {
			return fmt.Errorf("error while execing sql %v: %v", i, err)
		}
	}
	return nil
}

func getDBDeviceTx(tx *sqlx.Tx, extID types.ExternalDeviceID) (Device, bool) {
	var dev Device
	err := tx.Get(&dev, "SELECT * FROM device WHERE manufacturer=? AND external_id=? LIMIT 1", extID.Manufacturer, extID.ID)
	if err != nil || dev.ID == 0 {
		Log.Debug("could not get dbDevice", "err", err, "ext_id", extID, "dbDev_id", dev.ID)
		return dev, false
	}
	return dev, true
}

// A DeviceUpsertResponse describes the result of a call to UpsertDevice
type DeviceUpsertResponse struct {
	DeviceID           types.DeviceID
	UpsertedComponents map[string]types.Component
	DeletedComponents  map[string]types.Component
	HasDeviceChanged   bool
}

// UpsertDevice updates or inserts a Device into the SIFT database using the
// provided types.ExternalDeviceID. This also includes Upserting any Components
// which are attached to the Device. UpsertDevice considers the Device to be
// "whole"; if any Components which were previously attached to the Device are
// not present in the Device's Components map, those Components will be deleted
// from the SIFT database.
//
// If successful, the response object contains the new or updated Device's SIFT
// ID, as well as indications of which specific components were upserted or
// deleted.
func (sdb SiftDB) UpsertDevice(extID types.ExternalDeviceID, d types.Device) (resp DeviceUpsertResponse, err error) {
	sdb.log.Info("upserting device (incl. components)", "device", d)
	// Get a connection to the database
	db, err := sdb.DB()
	if err != nil {
		return DeviceUpsertResponse{}, err
	}
	defer db.Close()
	// begin a database transaction
	tx, err := db.Beginx()
	if err != nil {
		return DeviceUpsertResponse{}, fmt.Errorf("could not begin transaction: %v", err)
	}
	// If something bad happens, roll back the transaction
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				sdb.log.Error("could not roll back db transaction", "original_err", err, "rollback_err", rbErr)
			}
			sdb.log.Warn("rolled back db transaction", "original_err", err)
		} else {
			if cmErr := tx.Commit(); cmErr != nil {
				sdb.log.Error("could not commit transaction", "commit_err", cmErr)
				err = fmt.Errorf("could not commit transaction: %v", cmErr)
			}
			sdb.log.Debug("device updated; transaction committed")
		}
	}()

	// Upsert the base Device
	id, err := upsertDeviceTx(tx, extID, d)
	if err != nil {
		err = fmt.Errorf("could not upsert device: %v", err)
		return
	}

	var oldDevicePtr *types.Device
	// Compare the new device against the previous device
	// (or any empty device, if no previous device existed)
	oldDevice := types.Device{}
	if oldDevicePtr != nil {
		oldDevice = *oldDevicePtr
	}
	toUpsert, toDelete, hasDeviceChanged := lib.DiffDevice(oldDevice, d)

	// Upsert all new or updated components
	if err = upsertComponentsForDeviceTx(tx, id, toUpsert); err != nil {
		err = fmt.Errorf("could not upsert components for device %v: %v", id, err)
		return
	}

	// Delete any missing components
	for name := range toDelete {
		if err = deleteComponentTx(tx, id, name); err != nil {
			err = fmt.Errorf("could not delete component %v-%v: %v", id, name, err)
			return
		}
	}

	resp = DeviceUpsertResponse{
		DeviceID:           id,
		UpsertedComponents: toUpsert,
		DeletedComponents:  toDelete,
		HasDeviceChanged:   hasDeviceChanged,
	}
	return
}

type upsertDeviceResponse struct {
	id             string
	previousDevice *types.Device
	hasChanged     bool
}

func upsertDeviceTx(tx *sqlx.Tx, extID types.ExternalDeviceID, d types.Device) (types.DeviceID, error) {
	// Try updating
	q := "UPDATE device SET name=?, is_online=? WHERE manufacturer=? AND external_id=?"
	res, err := tx.Exec(q, d.Name, d.IsOnline, extID.Manufacturer, extID.ID)
	if err != nil {
		return 0, fmt.Errorf("error updating device: %v", err)
	}

	// Check the number of rows affected by the update; should be 1 if the
	// light_emitter_state row existed, and 0 if not
	if n, err := res.RowsAffected(); err != nil {
		return 0, fmt.Errorf("error getting row count (required for update): %v", err)
	} else if n == 0 {
		// The update failed, do an insert instead
		q = "INSERT INTO device (manufacturer, external_id, name, is_online) VALUES (?, ?, ?, ?)"
		res, err := tx.Exec(q, extID.Manufacturer, extID.ID, d.Name, d.IsOnline)
		if err != nil {
			return 0, fmt.Errorf("error inserting device: %v", err)
		}
		id, err := res.LastInsertId() // Get ID from insert
		if err != nil || id == 0 {
			return 0, fmt.Errorf("error or zero-value ID (id: %v, err: %v)", id, err)
		}
		Log.Debug("inserted new device", "id", id, "external_id", extID, "device", d, "query", q)
		return types.DeviceID(id), nil
	}

	// Do Select to get the ID
	var id int64
	if err = tx.Get(&id, "SELECT id FROM device WHERE manufacturer=? AND external_id=?", extID.Manufacturer, extID.ID); err != nil {
		return 0, fmt.Errorf("could not run select to get ID: %v", err)
	}

	Log.Debug("updated existing device", "id", id, "external_id", extID, "device", d, "query", q)
	return types.DeviceID(id), nil
}

func upsertComponentsForDeviceTx(tx *sqlx.Tx, id types.DeviceID, compsByName map[string]types.Component) error {
	Log.Debug("upserting components", "device_id", id, "components", compsByName, "tx", tx)
	for name, component := range compsByName {
		id, err := upsertComponentTx(tx, id, name, component)
		if err != nil || id == 0 {
			Log.Debug("could not upsert component", "err", err, "device_id", id, "comp_name", name, "comp", component)
			return fmt.Errorf("could not upsert component %v-%v: %v", id, name, err)
		}
	}
	return nil
}

// GetDevices returns a map of all Devices in the SIFT database, indexed by
// their SIFT-internal DeviceIDs, and expanded to the degree indicated by
// exFlags
func (sdb SiftDB) GetDevices(exFlags ExpansionFlags) (devs map[types.DeviceID]types.Device, err error) {
	// Get a connection to the database
	db, err := sdb.DB()
	if err != nil {
		return nil, fmt.Errorf("could not establish connection to database: %v", err)
	}
	defer db.Close()
	// begin a database transaction
	tx, err := db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %v", err)
	}
	// If something bad happens, roll back the transaction
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				sdb.log.Error("could not roll back db transaction", "original_err", err, "rollback_err", rbErr)
			}
			sdb.log.Warn("rolled back db transaction", "original_err", err)
		} else {
			if cmErr := tx.Commit(); cmErr != nil {
				sdb.log.Error("could not commit transaction", "commit_err", cmErr)
			}
			sdb.log.Debug("device updated; transaction committed")
		}
	}()
	devs, err = getDevicesTx(tx, exFlags)
	return
}

func getDevicesTx(tx *sqlx.Tx, exFlags ExpansionFlags) (map[types.DeviceID]types.Device, error) {
	devIDs := []int64{}
	if err := tx.Select(&devIDs, "SELECT id FROM device"); err != nil {
		return nil, fmt.Errorf("error getting device IDs from database: %v", err)
	}

	devs := make(map[types.DeviceID]types.Device)
	for _, id := range devIDs {
		dev := types.Device{}
		devID := types.DeviceID(id)
		if err := getDeviceTx(tx, &dev, devID, exFlags); err != nil {
			return nil, fmt.Errorf("could not get expanded Device with ID %v: %v", id, err)
		}
		devs[devID] = dev
	}
	return devs, nil
}

func (sdb SiftDB) getDevice(d *types.Device, id types.DeviceID, exFlags ExpansionFlags) (err error) {
	// Get a connection to the database
	db, err := sdb.DB()
	if err != nil {
		return err
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
				sdb.log.Error("could not roll back db transaction", "original_err", err, "rollback_err", rbErr)
			}
			sdb.log.Warn("rolled back db transaction", "original_err", err)
		} else {
			if cmErr := tx.Commit(); cmErr != nil {
				sdb.log.Error("could not commit transaction", "commit_err", cmErr)
			}
			sdb.log.Debug("device updated; transaction committed")
		}
	}()

	err = getDeviceTx(tx, d, id, exFlags)
	return
}

func getDeviceTx(tx *sqlx.Tx, d *types.Device, id types.DeviceID, exFlags ExpansionFlags) error {
	var dbDev Device
	q := "SELECT * FROM device WHERE id=?"
	if err := tx.Get(&dbDev, q, id); err != nil {
		return fmt.Errorf("could not get device: %v", err) // device was probably not found
	}

	// device was found in database, copy in the values
	if dbDev.Name.Valid {
		d.Name = dbDev.Name.String
	}
	d.IsOnline = dbDev.IsOnline

	// Populate the Components list for the Device
	if d.Components == nil {
		d.Components = make(map[string]types.Component)
	}
	if err := getComponentsForDevice(tx, d.Components, id, exFlags); err != nil {
		return fmt.Errorf("error getting components for Device: %v", err)
	}

	Log.Debug("got completed device from DB", "id", id, "device", d)
	return nil
}

func getComponentsForDevice(tx *sqlx.Tx, dest map[string]types.Component, deviceID types.DeviceID, exFlags ExpansionFlags) error {
	// get base components matching the device ID
	componentIDs := []int64{}
	err := tx.Select(&componentIDs, "SELECT id FROM component WHERE device_id=?", deviceID)
	if err != nil {
		return fmt.Errorf("error getting components for device %v: %v", deviceID, err)
	}

	Log.Debug("queried for components", "device_id", deviceID, "found_component_ids", componentIDs)

	// Build a set of components from database
	for _, id := range componentIDs {
		name, comp, err := getComponentTx(tx, id, exFlags)
		if err != nil {
			return fmt.Errorf("error getting component with ID %v: %v", id, err)
		}
		dest[name] = comp // put into the destination map
	}
	return nil // all components gathered without error
}

// GetComponents returns a map of all Components in the SIFT database, indexed
// by their SIFT-internal ComponentIDs, and expanded to the degree indicated by
// exFlags
func (sdb SiftDB) GetComponents(exFlags ExpansionFlags) (comps map[types.ComponentID]types.Component, err error) {
	// Get a connection to the database
	db, err := sdb.DB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	// begin a database transaction
	tx, err := db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %v", err)
	}
	// If something bad happens, roll back the transaction
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				sdb.log.Error("could not roll back db transaction", "original_err", err, "rollback_err", rbErr)
			}
			sdb.log.Warn("rolled back db transaction", "original_err", err)
		} else {
			if cmErr := tx.Commit(); cmErr != nil {
				sdb.log.Error("could not commit transaction", "commit_err", cmErr)
			}
			sdb.log.Debug("device updated; transaction committed")
		}
	}()

	comps, err = getAllComponents(tx, exFlags)
	sdb.log.Debug("got all components", "err", err, "comps", comps, "expand_flags", exFlags)
	return
}

func getAllComponents(tx *sqlx.Tx, exFlags ExpansionFlags) (map[types.ComponentID]types.Component, error) {
	type result struct {
		ID       int64
		DeviceID int64 `db:"device_id"`
	}
	// get results for each component in the database
	results := []result{}
	err := tx.Select(&results, "SELECT id, device_id FROM component")
	if err != nil {
		return nil, fmt.Errorf("error getting component IDs: %v", err)
	}

	// Build a set of components from database
	componentsByKey := make(map[types.ComponentID]types.Component)
	for _, result := range results {
		name, comp, err := getComponentTx(tx, result.ID, exFlags)
		if err != nil {
			return nil, fmt.Errorf("error getting component with ID %v: %v", result.ID, err)
		}
		key := types.ComponentID{
			Name:     name,
			DeviceID: types.DeviceID(result.DeviceID),
		}
		componentsByKey[key] = comp // put into the destination map
	}
	return componentsByKey, nil // all components gathered without error
}

func getComponentTx(tx *sqlx.Tx, id int64, exFlags ExpansionFlags) (string, types.Component, error) {
	// Get the db base component
	dbBaseComp, err := getBaseComponentByIDTx(tx, id)
	if err != nil {
		return "", nil, fmt.Errorf("could not get component with id %v: %v", id, err)
	}

	switch dbBaseComp.Type {
	default:
		return "", nil, fmt.Errorf("unhandled type: %v", dbBaseComp.Type)
	case types.LightEmitter{}.Type():
		le, err := getLightEmitterTx(tx, dbBaseComp, exFlags)
		return dbBaseComp.Name, le, err
	case types.MediaPlayer{}.Type():
		mp, err := getMediaPlayerTx(tx, dbBaseComp, exFlags)
		return dbBaseComp.Name, mp, err
	}
}

func getBaseComponentByIDTx(tx *sqlx.Tx, id int64) (Component, error) {
	var dbComp Component
	if err := tx.Get(&dbComp, "SELECT * FROM component WHERE id=? LIMIT 1", id); err != nil {
		Log.Debug("could not get base component", "id", id, "err", err)
		return Component{}, fmt.Errorf("error getting component with id %v: %v", id, err)
	}
	if dbComp.ID == 0 {
		Log.Warn("base component has id 0", "search_id", id)
		return Component{}, fmt.Errorf("got unexpected component id: 0")
	}
	Log.Debug("got base component", "id", id, "comp", dbComp)
	return dbComp, nil
}

func getBaseComponentTx(tx *sqlx.Tx, deviceID types.DeviceID, compName string) (Component, bool) {
	var dbComp Component
	err := tx.Get(&dbComp, "SELECT * FROM component WHERE device_id=? AND name=? LIMIT 1", deviceID, compName)
	if err != nil || dbComp.ID == 0 {
		return dbComp, false
	}
	return dbComp, true
}

func upsertBaseComponentTx(tx *sqlx.Tx, deviceID types.DeviceID, name string, c types.Component) (int64, error) {
	base := c.GetBaseComponent()
	existingBaseComp, found := getBaseComponentTx(tx, deviceID, name)
	if !found {
		// not found: do insert
		q := "INSERT INTO component (device_id, name, make, model, type) VALUES (?, ?, ?, ?, ?)"
		res, err := tx.Exec(q, deviceID, name, base.Make, base.Model, c.Type())
		if err != nil {
			return 0, fmt.Errorf("error inserting component: %v", err)
		}
		id, err := res.LastInsertId() // Get ID from insert
		if err != nil || id == 0 {
			return 0, fmt.Errorf("error or zero-value ID (id: %v, err: %v)", id, err)
		}
		Log.Debug("inserted component", "id", id, "base_component", base, "stmt", q)
		return id, nil
	}

	// found: do update
	q := "UPDATE component SET make=?, model=?, type=? WHERE id=?;"
	_, err := tx.Exec(q, base.Make, base.Model, c.Type(), existingBaseComp.ID)
	if err != nil {
		return 0, fmt.Errorf("error updating base component: %v", err)
	}
	Log.Debug("updated component", "base", base, "query", q, "update_err", err)
	return existingBaseComp.ID, err
}

func upsertComponentTx(tx *sqlx.Tx, deviceID types.DeviceID, compName string, comp types.Component) (id int64, err error) {
	// Upsert the base component and get comp ID
	id, err = upsertBaseComponentTx(tx, deviceID, compName, comp)
	if err != nil || id == 0 {
		return 0, err
	}

	// Upsert the specific component
	switch typed := comp.(type) {
	default:
		return 0, fmt.Errorf("unhandled component type: %T", comp)
	case types.LightEmitter:
		return id, upsertLightEmitterTx(tx, id, typed)
	case types.MediaPlayer:
		return id, upsertMediaPlayerTx(tx, id, typed)
	}
}

func deleteComponentTx(tx *sqlx.Tx, deviceID types.DeviceID, compName string) error {
	// Get the base component, if it exists
	if comp, found := getBaseComponentTx(tx, deviceID, compName); found {
		// delete the specific component info (changes by type)
		switch comp.Type {
		default:
			return fmt.Errorf("unhandled component type: %T", comp)
		case types.LightEmitter{}.Type():
			if err := deleteLightEmitterTx(tx, comp.ID); err != nil {
				return fmt.Errorf("could not delete light emitter: %v", err)
			}
		case types.MediaPlayer{}.Type():
			if err := deleteMediaPlayerTx(tx, comp.ID); err != nil {
				return fmt.Errorf("could not delete media player: %v", err)
			}
		}

		// delete the base component
		if _, err := tx.Queryx("DELETE FROM component WHERE device_id=? AND name=?", deviceID, compName); err != nil {
			return fmt.Errorf("error deleting base component: %v", err)
		}
	}
	return nil
}

type dbLightEmitter struct {
	Component
	types.LightEmitterState
	types.LightEmitterSpecs
	types.LightEmitterStats
}

func getLightEmitterTx(tx *sqlx.Tx, dbc Component, exFlags ExpansionFlags) (types.LightEmitter, error) {
	dbLE, err := getDBLightEmitterTx(tx, dbc.ID, dbc, exFlags)
	if err != nil {
		return types.LightEmitter{}, fmt.Errorf("error getting light emitter with id %v: %v", dbc.ID, err)
	}

	if dbLE.BrightnessInPercent < 0 || dbLE.BrightnessInPercent > 255 {
		return types.LightEmitter{}, fmt.Errorf("brightness value from database does not fit in uint8: %v", dbLE.BrightnessInPercent)
	}

	brightnessAsUint8 := uint8(dbLE.BrightnessInPercent)

	le := types.LightEmitter{
		BaseComponent: dbToBaseComponent(dbc),
		State: types.LightEmitterState{
			BrightnessInPercent: brightnessAsUint8,
		},
	}
	if exFlags&(ExpandAll|exFlags&ExpandSpecs) != 0 {
		le.Specs = &types.LightEmitterSpecs{
			MaxOutputInLumens:       dbLE.MaxOutputInLumens,
			MinOutputInLumens:       dbLE.MinOutputInLumens,
			ExpectedLifetimeInHours: dbLE.ExpectedLifetimeInHours,
		}
	}
	return le, nil
}

func getDBLightEmitterTx(tx *sqlx.Tx, id int64, baseComp Component, exFlags ExpansionFlags) (dbLightEmitter, error) {
	var dbLE dbLightEmitter

	// build a select statement based on the expand keys
	stmt := "SELECT * FROM component c JOIN light_emitter_state lstate ON c.id=lstate.id"

	if exFlags&(ExpandAll|ExpandSpecs) != 0 {
		stmt += " JOIN light_emitter_spec lspec ON lspec.make=c.make AND lspec.model=c.model"
	}

	// TODO: Add stats
	//	if exFlags&(ExpandAll|exFlags&ExpandStats) != 0 {
	//		stmt += " JOIN light_emitter_stats lstats ON c.id=lstats.id"
	//	}
	stmt += " WHERE c.id=? LIMIT 1"
	Log.Debug("getting light emitter", "query", stmt, "id", id)
	if err := tx.Get(&dbLE, stmt, id); err != nil {
		return dbLightEmitter{}, fmt.Errorf("error getting light emitter with id %v: %v", id, err)
	}

	if dbLE.ID == 0 {
		Log.Warn("light emitter has id 0", "search_id", id)
		return dbLightEmitter{}, fmt.Errorf("got unexpected component id: 0")
	}
	return dbLE, nil
}

// upsertLightEmitterTx upserts a light emitter to the database.
func upsertLightEmitterTx(tx *sqlx.Tx, compID int64, le types.LightEmitter) error {
	// Try updating
	q := "UPDATE light_emitter_state SET brightness_in_percent=? WHERE id=?"
	res, err := tx.Exec(q, le.State.BrightnessInPercent, compID)
	if err != nil {
		return fmt.Errorf("error updating component: %v", err)
	}

	// Check the number of rows affected by the udpate; should be 1 if the
	// light_emitter_state row existed, and 0 if not
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("error getting row count (required for update): %v", err)
	} else if n == 0 {
		// The update failed, do an insert instead
		q = "INSERT INTO light_emitter_state (id, brightness_in_percent) VALUES (?, ?)"
		res, err := tx.Exec(q, compID, le.State.BrightnessInPercent)
		if err != nil {
			return fmt.Errorf("error inserting component: %v", err)
		}
		id, err := res.LastInsertId() // Get ID from insert
		if err != nil || id == 0 {
			return fmt.Errorf("error or zero-value ID (id: %v, err: %v)", id, err)
		}
		Log.Debug("inserted new light emitter", "id", compID, "new_values", le, "query", q)
		return nil
	}
	Log.Debug("updated existing light emitter", "id", compID, "new_values", le, "query", q)
	return nil
}

func (sdb *SiftDB) expandLightEmitter(le *types.LightEmitter, exFlags ExpansionFlags) error {
	if exFlags&(ExpandAll|exFlags&ExpandSpecs) != 0 {
		if err := sdb.expandLightEmitterSpecs(le); err != nil {
			sdb.log.Debug("could not expand light emitter specs", "err", err, "light_emitter", le)
			return fmt.Errorf("could not expand light emitter specs: %v", err)
		}
		sdb.log.Debug("expanded specs for light emitter", "light_emitter", le)
	}
	if exFlags&(ExpandAll|exFlags&ExpandStats) != 0 {
		sdb.log.Warn("STUB: expandLightEmitter does not yet expand stats!")
	}
	return nil
}

func (sdb *SiftDB) expandLightEmitterSpecs(le *types.LightEmitter) error {
	specs, err := sdb.getLightEmitterSpecs(le.Make, le.Model)
	if err != nil {
		return err
	}
	le.Specs = specs
	return nil
}

func (sdb *SiftDB) getLightEmitterSpecs(make, model string) (*types.LightEmitterSpecs, error) {
	// Get a connection to the database
	db, err := sdb.DB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Run the query to get the appropriate specs
	spec := types.LightEmitterSpecs{}
	q := `SELECT max_output_in_lumens, min_output_in_lumens, expected_lifetime_in_hours
		FROM light_emitter_spec
		WHERE make=? AND model=?
		LIMIT 1`
	err = db.Get(&spec, q, make, model)
	if err != nil {
		return nil, fmt.Errorf("error querying for light emitter spec: %v", err)
	}
	sdb.log.Debug("got light emitter specs", "make", make, "model", model, "specs", spec)
	return &spec, nil
}

func deleteLightEmitterTx(tx *sqlx.Tx, compID int64) error {
	if _, err := tx.Queryx("DELETE FROM light_emitter_state WHERE id=?", compID); err != nil {
		return fmt.Errorf("error deleteing from light_emitter_state: %v", err)
	}
	//TODO: also delete light_emitter_stats
	return nil
}

//
// Media players
//
type dbMediaPlayer struct {
	Component
	types.MediaPlayerState
	types.MediaPlayerSpecs
	types.MediaPlayerStats
}

func getMediaPlayerTx(tx *sqlx.Tx, dbc Component, exFlags ExpansionFlags) (types.MediaPlayer, error) {
	dbMP, err := getDBMediaPlayerTx(tx, dbc.ID, dbc, exFlags)
	if err != nil {
		return types.MediaPlayer{}, fmt.Errorf("error getting media player with id %v: %v", dbc.ID, err)
	}

	mp := types.MediaPlayer{
		BaseComponent: dbToBaseComponent(dbc),
		State: types.MediaPlayerState{
			PlayState: dbMP.PlayState,
			MediaType: dbMP.MediaType,
			Source:    dbMP.Source,
		},
	}
	if exFlags&(ExpandAll|exFlags&ExpandSpecs) != 0 {
		mp.Specs = &types.MediaPlayerSpecs{
			SupportedAudioTypes: dbMP.SupportedAudioTypes,
			SupportedVideoTypes: dbMP.SupportedVideoTypes,
		}
	}
	return mp, nil
}

func getDBMediaPlayerTx(tx *sqlx.Tx, id int64, baseComp Component, exFlags ExpansionFlags) (dbMediaPlayer, error) {
	var dbMP dbMediaPlayer

	// build a select statement based on the expand keys
	stmt := "SELECT * FROM component c JOIN media_player_state mpstate ON c.id=mpstate.id"

	if exFlags&(ExpandAll|ExpandSpecs) != 0 {
		stmt += " JOIN media_player_spec mpspec ON mpspec.make=c.make AND mpspec.model=c.model"
	}

	// TODO: Add stats
	//	if exFlags&(ExpandAll|exFlags&ExpandStats) != 0 {
	//		stmt += " JOIN media_player_stats lstats ON c.id=lstats.id"
	//	}
	stmt += " WHERE c.id=? LIMIT 1"
	Log.Debug("getting media player", "query", stmt, "id", id)
	if err := tx.Get(&dbMP, stmt, id); err != nil {
		return dbMediaPlayer{}, fmt.Errorf("error getting media player with id %v: %v", id, err)
	}

	if dbMP.ID == 0 {
		Log.Warn("media player has id 0", "search_id", id)
		return dbMediaPlayer{}, fmt.Errorf("got unexpected component id: 0")
	}
	return dbMP, nil
}

// upsertMediaPlayerTx upserts a media player to the database.
func upsertMediaPlayerTx(tx *sqlx.Tx, compID int64, mp types.MediaPlayer) error {
	// Try updating
	q := "UPDATE media_player_state SET play_state=?, media_type=?, source=? WHERE id=?"
	res, err := tx.Exec(q, mp.State.PlayState, mp.State.MediaType, mp.State.Source, compID)
	if err != nil {
		return fmt.Errorf("error updating component: %v", err)
	}

	// Check the number of rows affected by the udpate; should be 1 if the
	// media_player_state row existed, and 0 if not
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("error getting row count (required for update): %v", err)
	} else if n == 0 {
		// The update failed, do an insert instead
		q = "INSERT INTO media_player_state (id, play_state, media_type, source) VALUES (?, ?, ?, ?)"
		res, err := tx.Exec(q, compID, mp.State.PlayState, mp.State.MediaType, mp.State.Source)
		if err != nil {
			return fmt.Errorf("error inserting component: %v", err)
		}
		id, err := res.LastInsertId() // Get ID from insert
		if err != nil || id == 0 {
			return fmt.Errorf("error or zero-value ID (id: %v, err: %v)", id, err)
		}
		Log.Debug("inserted new media player", "id", compID, "new_values", mp, "query", q)
		return nil
	}
	Log.Debug("updated existing media player", "id", compID, "new_values", mp, "query", q)
	return nil
}

func (sdb *SiftDB) expandMediaPlayer(le *types.MediaPlayer, exFlags ExpansionFlags) error {
	if exFlags&(ExpandAll|exFlags&ExpandSpecs) != 0 {
		if err := sdb.expandMediaPlayerSpecs(le); err != nil {
			sdb.log.Debug("could not expand media player specs", "err", err, "media_player", le)
			return fmt.Errorf("could not expand media player specs: %v", err)
		}
		sdb.log.Debug("expanded specs for media player", "media_player", le)
	}
	if exFlags&(ExpandAll|exFlags&ExpandStats) != 0 {
		sdb.log.Warn("STUB: expandMediaPlayer does not yet expand stats!")
	}
	return nil
}

func (sdb *SiftDB) expandMediaPlayerSpecs(le *types.MediaPlayer) error {
	specs, err := sdb.getMediaPlayerSpecs(le.Make, le.Model)
	if err != nil {
		return err
	}
	le.Specs = specs
	return nil
}

func (sdb *SiftDB) getMediaPlayerSpecs(make, model string) (*types.MediaPlayerSpecs, error) {
	// Get a connection to the database
	db, err := sdb.DB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Run the query to get the appropriate specs
	spec := types.MediaPlayerSpecs{}
	q := `SELECT supported_audio_types, supported_video_types
		FROM media_player_spec
		WHERE make=? AND model=?
		LIMIT 1`
	err = db.Get(&spec, q, make, model)
	if err != nil {
		return nil, fmt.Errorf("error querying for media player spec: %v", err)
	}
	sdb.log.Debug("got media player specs", "make", make, "model", model, "specs", spec)
	return &spec, nil
}

func deleteMediaPlayerTx(tx *sqlx.Tx, compID int64) error {
	if _, err := tx.Queryx("DELETE FROM media_player_state WHERE id=?", compID); err != nil {
		return fmt.Errorf("error deleteing from media_player_state: %v", err)
	}
	//TODO: also delete media_player_stats
	return nil
}

//func (sdb *SiftDB) UpsertAdapterCredentials(key, value string) (err error) {
//	// Get a connection to the database
//	db, err := sdb.DB()
//	if err != nil {
//		return fmt.Errorf("error getting db connection: %v", err)
//	}
//	defer db.Close()
//	// begin a database transaction
//	tx, err := db.Beginx()
//	if err != nil {
//		return fmt.Errorf("could not begin transaction: %v", err)
//	}
//	// If something bad happens, roll back the transaction
//	defer func() {
//		if err != nil {
//			if rbErr := tx.Rollback(); rbErr != nil {
//				sdb.log.Error("could not roll back db transaction", "original_err", err, "rollback_err", rbErr)
//			}
//			sdb.log.Warn("rolled back db transaction", "original_err", err)
//		} else {
//			if cmErr := tx.Commit(); cmErr != nil {
//				sdb.log.Error("could not commit transaction", "commit_err", cmErr)
//				err = fmt.Errorf("could not commit transaction: %v", cmErr)
//			}
//			sdb.log.Debug("adapter credentials updated; transaction committed")
//		}
//	}()
//
//	// Try updating the Adapter Credentials
//	q := "UPDATE adapter_credentials SET value=? WHERE key=?"
//	res, err := tx.Exec(q, value, key)
//	if err != nil {
//		err = fmt.Errorf("error updating adapter_credentials: %v", err)
//		return
//	}
//
//	// Check the number of rows affected by the update; should be 1 if the
//	// row existed, and 0 if not
//	var n int64
//	if n, err = res.RowsAffected(); err != nil {
//		err = fmt.Errorf("error getting row count (required for update): %v", err)
//		return
//	} else if n == 0 {
//		// The update failed, do an insert instead
//		q = "INSERT INTO adapter_credentials (key, value) VALUES (?, ?)"
//		_, err = tx.Exec(q, key, value)
//		if err != nil {
//			err = fmt.Errorf("error inserting adapter credentials: %v", err)
//			return
//		}
//	}
//	return
//}
//
//func (sdb *SiftDB) GetAdapterCredentials(key string) (string, error) {
//	var value string
//	// Get a connection to the database
//	db, err := sdb.DB()
//	if err != nil {
//		return "", fmt.Errorf("error getting db connection: %v", err)
//	}
//	defer db.Close()
//	if err := db.Get(&value, "SELECT value FROM adapter_credentials WHERE key=?", key); err != nil {
//		return "", fmt.Errorf("could not get credentials from database: %v", err)
//	}
//	return value, nil
//}

//
// Helper functions
//

// dbToBaseComponent converts the internal database version of a base component
// into a types.BaseComponent. Note that some fields (like database IDs, the
// component name, and the component type) are not part of the
// types.BaseComponent structure because that information is not relevant, or
// is supplied by elsewhere (component name, for example, is held by the Device
// containing the Component)
func dbToBaseComponent(dbc Component) types.BaseComponent {
	return types.BaseComponent{
		Make:  dbc.Make,
		Model: dbc.Model,
	}
}

func (sdb *SiftDB) expandDevice(d *types.Device, exFlags ExpansionFlags) error {
	// Expand each component individually
	for name, comp := range d.Components {
		var err error
		var expandedComp types.Component // holds the expanded component
		switch typed := comp.(type) {    // The expansion process depends on the component type
		default:
			err = fmt.Errorf("unknown component type %T", comp)
		case types.LightEmitter:
			err = sdb.expandLightEmitter(&typed, exFlags)
			expandedComp = typed
			sdb.log.Debug("expanded light emitter component", "err", err, "comp", typed)
		}
		if err != nil {
			return fmt.Errorf("could not expand component %v: %v", name, err)
		}
		// if no errors occured, replace the expanded component into the map
		d.Components[name] = expandedComp
	}
	return nil
}

// GetExternalDeviceID determines the types.ExternalDeviceID that matches the
// given SIFT-internal types.DeviceID in the SIFT database.
func (sdb *SiftDB) GetExternalDeviceID(id types.DeviceID) (types.ExternalDeviceID, error) {
	// Get a connection to the database
	db, err := sdb.DB()
	if err != nil {
		return types.ExternalDeviceID{}, err
	}
	defer db.Close()
	dbDev := Device{}
	q := `SELECT * FROM device WHERE id = ? LIMIT 1`
	if err := db.Get(&dbDev, q, id); err != nil {
		return types.ExternalDeviceID{}, fmt.Errorf("could not get device from database: %v", err)
	}
	externalID := types.ExternalDeviceID{
		Manufacturer: dbDev.Manufacturer,
		ID:           dbDev.ExternalID,
	}
	sdb.log.Debug("got external ID for device", "id", id, "external_id", externalID)
	return externalID, nil
}

// markAllDevicesInactive will set is_online=false for all rows in the devices
// table of the SIFT DB.
func (sdb *SiftDB) markAllDevicesInactive() (err error) {
	// Get a connection to the database
	db, err := sdb.DB()
	if err != nil {
		return err
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
				sdb.log.Error("could not roll back db transaction", "original_err", err, "rollback_err", rbErr)
			}
			sdb.log.Warn("rolled back db transaction", "original_err", err)
		} else {
			if cmErr := tx.Commit(); cmErr != nil {
				sdb.log.Error("could not commit transaction", "commit_err", cmErr)
			}
			sdb.log.Debug("device updated; transaction committed")
		}
	}()

	_, err = db.Exec("UPDATE device SET is_online=?", false)
	return
}
