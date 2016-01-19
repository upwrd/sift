package lib

import (
	. "gopkg.in/check.v1"
	"github.com/upwrd/sift/types"
)

/*
func (s *TestSIFTLibSuite) TestDefaultPriorityDiffer(c *C) {
	pdiff := NewPrioritizer(nil)
	c.Assert(pdiff, NotNil)

	// Consider a DeviceUpdate - should be ignored since SetOutput has not been called
	noopUpdate := DeviceUpdated{ID: "device 1"}
	pdiff.Consider("adapter 1", noopUpdate)

	dest := make(chan interface{}, 10)
	err := pdiff.SetOutput(nil) // invalid channel, should error
	c.Assert(err, ErrorMatches, "destination channel cannot be nil")

	err = pdiff.SetOutput(dest) // valid channel, should not error
	c.Assert(err, IsNil)

	err = pdiff.SetOutput(dest) // already set, should error
	c.Assert(err, ErrorMatches, "output already set.*")

	// Consider an update to device 1 from adapter 1
	// Since no adapters have produced updates for device 1, this should produce an update
	light1 := types.LightEmitter{
		State: types.LightEmitterState{
			BrightnessInPercent: 100,
		},
	}
	device1 := types.Device{
		Components: map[string]types.Component{
			"light 1": light1,
		},
	}
	dev1Update := DeviceUpdated{
		ID:       "device 1",
		NewState: device1,
	}
	c.Assert(len(dest), Equals, 0)
	pdiff.Consider("adapter 1", dev1Update)
	c.Assert(len(dest), Equals, 1)
	fromChan := <-dest
	c.Assert(fromChan, DeepEquals, dev1Update)

	// Submit another device update from adapter 1.
	// This should also produce an update
	light1State2 := types.LightEmitter{
		State: types.LightEmitterState{
			BrightnessInPercent: 55,
		},
	}
	device1State2 := types.Device{
		Components: map[string]types.Component{
			"light 1": light1State2,
		},
	}
	dev1State2Update := DeviceUpdated{
		ID:       "device 1",
		NewState: device1State2,
	}
	c.Assert(len(dest), Equals, 0)
	pdiff.Consider("adapter 1", dev1State2Update)
	c.Assert(len(dest), Equals, 1)
	fromChan = <-dest
	c.Assert(fromChan, DeepEquals, dev1State2Update)

	// Consider an update to device 1 from adapter 3.
	// Since adapter 1 is higher priority (alphabetical), this should be ignored
	c.Assert(len(dest), Equals, 0)
	pdiff.Consider("adapter 3", dev1Update)
	c.Assert(len(dest), Equals, 0)

	// Consider an update to device 2 from adapter 3.
	// Since no adapters have produced updates for device 2, this should produce an update
	light2 := types.LightEmitter{
		State: types.LightEmitterState{
			BrightnessInPercent: 42,
		},
	}
	device2 := types.Device{
		Components: map[string]types.Component{
			"light 2": light2,
		},
	}
	dev2Update := DeviceUpdated{
		ID:       "device 2",
		NewState: device2,
	}
	c.Assert(len(dest), Equals, 0)
	pdiff.Consider("adapter 3", dev2Update)
	c.Assert(len(dest), Equals, 1)
	fromChan = <-dest
	c.Assert(fromChan, DeepEquals, dev2Update)

	// Consider an update to device 2 from adapter 2.
	// Since adapter 2 is higher priority (alphabetical), this should produce an update
	light2Ad2 := types.LightEmitter{
		State: types.LightEmitterState{
			BrightnessInPercent: 77,
		},
	}
	device2Ad2 := types.Device{
		Components: map[string]types.Component{
			"light 2": light2Ad2,
		},
	}
	dev2Ad2Update := DeviceUpdated{
		ID:       "device 2",
		NewState: device2Ad2,
	}
	c.Assert(len(dest), Equals, 0)
	pdiff.Consider("adapter 2", dev2Ad2Update)
	c.Assert(len(dest), Equals, 1)
	fromChan = <-dest
	c.Assert(fromChan, DeepEquals, dev2Ad2Update)

	// After considering device 2 from adapter 2, updates from adapter 3 should be ignored
	c.Assert(len(dest), Equals, 0)
	pdiff.Consider("adapter 3", dev2Update)
	c.Assert(len(dest), Equals, 0)

	// Delete device 1 from adapter 3
	// Since adapter 1 has been considered for device 1, and has higher priority (alphabetical),
	// this should produce no update
	dev1Delete := DeviceDeleted{
		ID: "device 1",
	}
	c.Assert(len(dest), Equals, 0)
	pdiff.Consider("adapter 3", dev1Delete)
	c.Assert(len(dest), Equals, 0)

	// Delete device 1 from adapter 2
	// Since adapter 2 was never considered for device 1, this should produce no update
	c.Assert(len(dest), Equals, 0)
	pdiff.Consider("adapter 2", dev1Delete)
	c.Assert(len(dest), Equals, 0)

	// Delete device 1 from adapter 1
	// Since adapter 1 has highest priority for device 1, this should produce an update
	c.Assert(len(dest), Equals, 0)
	pdiff.Consider("adapter 1", dev1Delete)
	c.Assert(len(dest), Equals, 1)
	fromChan = <-dest
	c.Assert(fromChan, DeepEquals, dev1Delete)

	// Consider an update to device 1 from adapter 3.
	// Since adapter 1 has indicated a delete (which should cause it to be removed from
	// consideration for device 1, this should produce an update
	c.Assert(len(dest), Equals, 0)
	pdiff.Consider("adapter 3", dev1Update)
	c.Assert(len(dest), Equals, 1)
	fromChan = <-dest
	c.Assert(fromChan, DeepEquals, dev1Update)
}
*/

type sortTest struct {
	sortFns       []lessFunc
	input, output []AdapterDescription
}

var sortTests = []sortTest{
	{
		sortFns: []lessFunc{byID},
		input: []AdapterDescription{
			{Type: ControllerTypeBluetooth, ID: "4"},
			{Type: ControllerTypeBluetooth, ID: "3"},
			{Type: ControllerTypeBluetooth, ID: "2"},
			{Type: ControllerTypeBluetooth, ID: "1"},
		},
		output: []AdapterDescription{
			{Type: ControllerTypeBluetooth, ID: "1"},
			{Type: ControllerTypeBluetooth, ID: "2"},
			{Type: ControllerTypeBluetooth, ID: "3"},
			{Type: ControllerTypeBluetooth, ID: "4"},
		},
	},
	{
		sortFns: []lessFunc{byType, byID},
		input: []AdapterDescription{
			{Type: ControllerTypeIPv4, ID: "1"},
			{Type: ControllerTypeIPv4, ID: "2"},
			{Type: ControllerTypeBluetooth, ID: "3"},
			{Type: ControllerTypeBluetooth, ID: "4"},
			{Type: ControllerTypeZWave, ID: "5"},
			{Type: ControllerTypeZWave, ID: "6"},
			{Type: ControllerTypeZigbee, ID: "7"},
			{Type: ControllerTypeZigbee, ID: "8"},
		},
		output: []AdapterDescription{
			{Type: ControllerTypeZigbee, ID: "7"},
			{Type: ControllerTypeZigbee, ID: "8"},
			{Type: ControllerTypeZWave, ID: "5"},
			{Type: ControllerTypeZWave, ID: "6"},
			{Type: ControllerTypeBluetooth, ID: "3"},
			{Type: ControllerTypeBluetooth, ID: "4"},
			{Type: ControllerTypeIPv4, ID: "1"},
			{Type: ControllerTypeIPv4, ID: "2"},
		},
	},
}

func (s *TestSIFTLibSuite) TestSorting(c *C) {
	for i, test := range sortTests {
		orderedBy(test.sortFns...).sort(test.input)
		c.Assert(test.input, DeepEquals, test.output, Commentf("failed test %v", i+1))
		//fmt.Printf("DEBUG:\n... input: %v\n...output: %v\n", test.input, test.output)
	}
}

type considerTestStep struct {
	desc                           AdapterDescription
	update                         interface{}
	expectedUpdate, expectedDelete interface{}
	expectedErr                    string
}

func sameProtocolTest() []considerTestStep {
	desc1 := AdapterDescription{
		Type: ControllerTypeZigbee,
		ID:   "1",
	}
	desc2 := AdapterDescription{
		Type: ControllerTypeZigbee,
		ID:   "2",
	}

	light1State1 := types.LightEmitter{
		State: types.LightEmitterState{
			BrightnessInPercent: 100,
		},
	}
	dev1State1 := types.Device{
		Components: map[string]types.Component{
			"light1": light1State1,
		},
	}
	id1 := types.ExternalDeviceID{Manufacturer: "foo", ID: "device1"}
	id2 := types.ExternalDeviceID{Manufacturer: "foo", ID: "device2"}

	return []considerTestStep{
		// consider device1 from desc1: should produce an update (no other contenders for device1)
		considerTestStep{
			desc: desc1,
			update: DeviceUpdated{
				ID:       id1,
				NewState: dev1State1,
			},
			expectedUpdate: DeviceUpdated{
				ID:       id1,
				NewState: dev1State1,
			},
		},
		// consider device1 from desc1 (again): should produce an update (no other contenders for device1)
		considerTestStep{
			desc: desc1,
			update: DeviceUpdated{
				ID:       id1,
				NewState: dev1State1,
			},
			expectedUpdate: DeviceUpdated{
				ID:       id1,
				NewState: dev1State1,
			},
		},
		// consider device1 from desc2: should NOT produce an update (device1 has higher priority)
		considerTestStep{
			desc: desc2,
			update: DeviceUpdated{
				ID:       id1,
				NewState: dev1State1,
			},
		},
		// consider device2 from desc2: should produce an update (no other contenders for device2)
		considerTestStep{
			desc: desc2,
			update: DeviceUpdated{
				ID:       id2,
				NewState: dev1State1,
			},
			expectedUpdate: DeviceUpdated{
				ID:       id2,
				NewState: dev1State1,
			},
		},
		// consider device2 from desc1: should produce an update (highest-ranked contender for device2)
		considerTestStep{
			desc: desc1,
			update: DeviceUpdated{
				ID:       id2,
				NewState: dev1State1,
			},
			expectedUpdate: DeviceUpdated{
				ID:       id2,
				NewState: dev1State1,
			},
		},
		// consider delete of device1 from desc1: should produce an update (highest-ranked contender for device1)
		considerTestStep{
			desc: desc1,
			update: DeviceDeleted{
				ID: id1,
			},
			expectedUpdate: DeviceDeleted{
				ID: id1,
			},
		},
		// consider device1 from desc2: should produce an update (no other contenders for device1, since desc1 has deleted)
		considerTestStep{
			desc: desc2,
			update: DeviceUpdated{
				ID:       id1,
				NewState: dev1State1,
			},
			expectedUpdate: DeviceUpdated{
				ID:       id1,
				NewState: dev1State1,
			},
		},
		// consider delete of device2 from desc2: should NOT produce an update (device1 has higher priority)
		considerTestStep{
			desc: desc2,
			update: DeviceDeleted{
				ID: id2,
			},
		},
	}
}

var considerTests = [][]considerTestStep{
	sameProtocolTest(),
}

func (s *TestSIFTLibSuite) TestConsider(c *C) {
	for _, test := range considerTests {
		byTypeByID := []lessFunc{byType, byID}
		p := NewPrioritizer(byTypeByID)
		updates := p.OutputChan()
		for i, step := range test {
			err := p.Consider(step.desc, step.update)
			if step.expectedErr != "" {
				c.Assert(err, ErrorMatches, step.expectedErr)
				continue
			}
			c.Assert(err, IsNil)

			if step.expectedUpdate != nil {
				c.Assert(len(updates), Equals, 1, Commentf("step %v (%+v)", i, step))
				update := <-updates
				c.Assert(update, DeepEquals, step.expectedUpdate, Commentf("step %v (%+v)", i, step))
			}
			if step.expectedDelete != nil {
				c.Assert(len(updates), Equals, 1, Commentf("step %v (%+v)", i, step))
				update := <-updates
				c.Assert(update, DeepEquals, step.expectedDelete, Commentf("step %v (%+v)", i, step))
			}
			if step.expectedUpdate == nil && step.expectedDelete == nil {
				c.Assert(len(updates), Equals, 0, Commentf("step %v (%+v)", i, step))
			}
		}
	}
}
