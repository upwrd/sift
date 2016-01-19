package db

import (
	. "gopkg.in/check.v1"
	"github.com/upwrd/sift/types"
	"sync"
	"testing"

	// "github.com/upwrd/sift/logging"
)

// Hook up gocheck into the "go test" runner.
func TestDB(t *testing.T) { TestingT(t) }

type DBTestSuite struct{}

var _ = Suite(&DBTestSuite{})

func (s *DBTestSuite) TestNew(c *C) {
	db, err := Open("")
	c.Assert(err, IsNil)
	c.Assert(db, NotNil)
	err = db.Close()
	c.Assert(err, IsNil)
}

type dbTest struct {
	obj         interface{}
	key         string
	expectedErr string
}

type insertAndGetTest struct {
	id          types.ExternalDeviceID
	device      types.Device
	expectedErr string
}

func upsertAndGetTests() []insertAndGetTest{
	return []insertAndGetTest{
		// Should fail
		{
			// nil device should fail
			expectedErr: ".*CHECK constraint failed.*",
		},
		{
			// Manufacturer and External ID are empty
			device: types.Device{
				Name:     "Kitchen Light",
				IsOnline: true,
				Components: map[string]types.Component{
					"bulb_v1": types.LightEmitter{
						BaseComponent: types.BaseComponent{
							Make:  "example",
							Model: "light_emitter_1",
						},
						State: types.LightEmitterState{
							BrightnessInPercent: uint8(55),
						},
					},
				},
			},
			expectedErr: ".*CHECK constraint failed.*",
		},
		// Should succeed
		{
			id: types.ExternalDeviceID{
				Manufacturer: "upward",
				ID:           "0001ab",
			},
			device: types.Device{
				Name:     "foo",
				IsOnline: true,
				Components: map[string]types.Component{
					"bulb_v1": types.LightEmitter{
						BaseComponent: types.BaseComponent{
							Make:  "example",
							Model: "light_emitter_1",
						},
						State: types.LightEmitterState{
							BrightnessInPercent: uint8(55),
						},
					},
				},
			},
		},
		// Should succeed by overwriting previous
		{
			id: types.ExternalDeviceID{
				Manufacturer: "upward",
				ID:           "0001ab",
			},
			device: types.Device{
				Name:     "bar",
				IsOnline: true,
				Components: map[string]types.Component{
					"bulb_v1": types.LightEmitter{
						BaseComponent: types.BaseComponent{
							Make:  "example",
							Model: "light_emitter_1",
						},
						State: types.LightEmitterState{
							BrightnessInPercent: uint8(55),
						},
					},
				},
			},
		},
		// Should succeed
		{
			id: types.ExternalDeviceID{
				Manufacturer: "upward",
				ID:           "ba1000",
			},
			device: types.Device{
				Name:     "foo",
				IsOnline: true,
				Components: map[string]types.Component{
					"bulb_v1": types.LightEmitter{
						BaseComponent: types.BaseComponent{
							Make:  "example",
							Model: "light_emitter_1",
						},
						State: types.LightEmitterState{
							BrightnessInPercent: uint8(55),
						},
					},
				},
			},
		},
		// Should succeed by overwriting previous
		{
			id: types.ExternalDeviceID{
				Manufacturer: "upward",
				ID:           "ba1000",
			},
			device: types.Device{
				Name:     "bar",
				IsOnline: true,
				Components: map[string]types.Component{
					"bulb_v1": types.LightEmitter{
						BaseComponent: types.BaseComponent{
							Make:  "example",
							Model: "light_emitter_1",
						},
						State: types.LightEmitterState{
							BrightnessInPercent: uint8(55),
						},
					},
				},
			},
		},
	}
}

// TestInsertAndGet tests inserting a list of Devices to the database, then retrieving them
func (s *DBTestSuite) TestInsertAndGet(c *C) {
	//logging.SetLevelStr("debug")
	db, err := Open("") // "" indicates temporary file; replace with filename to inspect using sqlite
	c.Assert(err, IsNil)
	c.Assert(db, NotNil)
	defer db.Close()

	for i, test := range upsertAndGetTests() {
		// store the Device
		resp, err := db.UpsertDevice(test.id, test.device)
		if test.expectedErr != "" {
			c.Assert(err, ErrorMatches, test.expectedErr)
			continue
		}
		c.Assert(err, IsNil)
		c.Assert(resp.DeviceID, Not(Equals), 0)

		// get the Device back from db
		var fromDB types.Device
		err = db.getDevice(&fromDB, resp.DeviceID, ExpandNone)
		c.Assert(err, IsNil)
		c.Assert(fromDB, DeepEquals, test.device, Commentf("test %v", i))

		// get the Device with expanded Component specs
		err = db.getDevice(&fromDB, resp.DeviceID, ExpandSpecs)
		c.Assert(err, IsNil)
		c.Assert(fromDB.Name, Equals, test.device.Name)
		c.Assert(len(fromDB.Components), Equals, len(test.device.Components))
		c.Assert(fromDB, Not(DeepEquals), test.device)
		err = db.expandDevice(&test.device, ExpandSpecs) // manually expand the device with component specs
		c.Assert(err, IsNil)
		c.Assert(fromDB, DeepEquals, test.device)
	}
}

func (s *DBTestSuite) TestInsertAndGetStress(c *C) {
	wg := sync.WaitGroup{}
	for i := 0; i < 50; i++ {
		go func() {
			wg.Add(1)
			s.TestInsertAndGet(c)
			wg.Done()
		}()
	}
	wg.Wait()
}

func (s *DBTestSuite) TestGetComponents(c *C) {
	// logging.SetLevelStr("debug")
	db, err := Open("test.db") // "" indicates temporary file; replace with filename to inspect using sqlite
	c.Assert(err, IsNil)
	c.Assert(db, NotNil)
	defer db.Close()

	id := types.ExternalDeviceID{
		Manufacturer: "upward",
		ID:           "0001ab",
	}
	dev := types.Device{
		Name:     "Kitchen Light",
		IsOnline: true,
		Components: map[string]types.Component{
			"bulb_v1": types.LightEmitter{
				BaseComponent: types.BaseComponent{
					Make:  "example",
					Model: "light_emitter_1",
				},
				State: types.LightEmitterState{
					BrightnessInPercent: uint8(55),
				},
			},
		},
	}

	// store the Device
	resp, err := db.UpsertDevice(id, dev)
	c.Assert(err, IsNil)
	c.Assert(resp.DeviceID, Not(Equals), 0)

	// get all components
	comps, err := db.GetComponents(ExpandNone)
	c.Assert(err, IsNil)
	c.Assert(len(comps), Equals, 1)

	key := types.ComponentID{DeviceID: resp.DeviceID, Name: "bulb_v1"}
	bulbv1, ok := comps[key]
	c.Assert(ok, Equals, true)
	bulbv1Expected := types.LightEmitter{
		BaseComponent: types.BaseComponent{
			Make:  "example",
			Model: "light_emitter_1",
		},
		State: types.LightEmitterState{
			BrightnessInPercent: uint8(55),
		},
	}
	c.Assert(bulbv1, DeepEquals, bulbv1Expected)
}
