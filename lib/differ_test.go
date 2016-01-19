package lib

import (
	. "gopkg.in/check.v1"
	"math/rand"
	"sift/types"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func TestSIFTLib(t *testing.T) { TestingT(t) }

type TestSIFTLibSuite struct{}

var _ = Suite(&TestSIFTLibSuite{})

func (s *TestSIFTLibSuite) TestAllAtOnceDiffer(c *C) {
	differ := NewAllAtOnceDiffer()
	c.Assert(differ, NotNil)
	var badDest chan interface{}
	err := differ.SetOutput(badDest) // not instantiated, should error
	c.Assert(err, NotNil)

	// Consider a simple device set - should indicate an update
	device0 := types.Device{Components: map[string]types.Component{}}
	key0 := types.ExternalDeviceID{Manufacturer: "foo", ID: "device 0"}
	devices0 := map[types.ExternalDeviceID]types.Device{key0: device0}
	differ.Consider(devices0) // should be ignored since SetOutput had not been called

	dest := make(chan interface{}, 10)
	err = differ.SetOutput(dest) // valid channel, should not error
	c.Assert(err, IsNil)

	err = differ.SetOutput(dest) // already set, should error
	c.Assert(err, ErrorMatches, "output already set.*")

	// Consider an empty device set - should not indicate a change
	devices := map[types.ExternalDeviceID]types.Device{}
	differ.Consider(devices)
	c.Assert(len(dest), Equals, 0)

	// Consider a simple device set - should indicate an update
	lightEmitter1 := types.LightEmitter{}
	device1 := types.Device{
		Components: map[string]types.Component{
			"light emitting component 1": lightEmitter1,
		},
	}
	key1 := types.ExternalDeviceID{Manufacturer: "foo", ID: "device 1"}
	devices1 := map[types.ExternalDeviceID]types.Device{key1: device1}
	differ.Consider(devices1)
	c.Assert(len(dest), Equals, 1)
	update := <-dest
	expected := DeviceUpdated{
		ID:      key1,
		NewState: device1,
	}
	c.Assert(update, DeepEquals, expected)

	// Consider an identical device set - should not indicate a change
	for i := 0; i < 100; i++ {
		lightEmitter1Copy := types.LightEmitter{}
		device1Copy := types.Device{
			Components: map[string]types.Component{
				"light emitting component 1": lightEmitter1Copy,
			},
		}
		devices2 := map[types.ExternalDeviceID]types.Device{key1: device1Copy}
		differ.Consider(devices2)
		c.Assert(len(dest), Equals, 0)
	}

	// Consider a slightly-changed device set - should indicate an update
	lightEmitter1Updated := types.LightEmitter{
		State: types.LightEmitterState{
			BrightnessInPercent: 100,
		},
	}
	device1Updated := types.Device{
		Components: map[string]types.Component{
			"light emitting component 1": lightEmitter1Updated,
		},
	}
	devices4 := map[types.ExternalDeviceID]types.Device{key1: device1Updated}
	differ.Consider(devices4)
	c.Assert(len(dest), Equals, 1)
	update = <-dest
	expected = DeviceUpdated{
		ID:      key1,
		NewState: device1Updated,
	}
	c.Assert(update, DeepEquals, expected)

	// Consider an empty device set - should indicate a deletion
	devices5 := map[types.ExternalDeviceID]types.Device{}
	differ.Consider(devices5)
	c.Assert(len(dest), Equals, 1)
	update = <-dest
	expectedDelete := DeviceDeleted{
		ID: key1,
	}
	c.Assert(update, DeepEquals, expectedDelete)
}

func (s *TestSIFTLibSuite) TestGetLatest(c *C) {
	differ := NewAllAtOnceDiffer()
	c.Assert(differ, NotNil)
	dest := make(chan interface{}, 10)
	err := differ.SetOutput(dest) // valid channel, should not error
	c.Assert(err, IsNil)

	// Consider a simple device set - should indicate an update
	lightEmitter1 := types.LightEmitter{}
	device1 := types.Device{
		Components: map[string]types.Component{
			"light emitting component 1": lightEmitter1,
		},
	}
	id1 := types.ExternalDeviceID{Manufacturer: "foo", ID: "device 1"}
	devices := map[types.ExternalDeviceID]types.Device{id1: device1}
	differ.Consider(devices)
	c.Assert(len(dest), Equals, 1)
	update := <-dest
	expected := DeviceUpdated{
		ID:      id1,
		NewState: device1,
	}
	c.Assert(update, DeepEquals, expected)

	latest, err := differ.GetLatest(id1)
	c.Assert(err, IsNil)
	c.Assert(latest, DeepEquals, device1)

	badID := types.ExternalDeviceID{Manufacturer: "foo", ID: "bar"}
	_, err = differ.GetLatest(badID)
	c.Assert(err, ErrorMatches, "cannot find device with key.*")
}

func (s *TestSIFTLibSuite) TestSynchronousDiff(c *C) {
	differ := NewAllAtOnceDiffer()
	c.Assert(differ, NotNil)
	dest := make(chan interface{}, 10)
	err := differ.SetOutput(dest) // valid channel, should not error
	c.Assert(err, IsNil)

	var lastBrightness = uint32(0)
	for i := 0; i < 100; i++ {

		randBrightness := rand.Uint32() % 100
		if randBrightness != lastBrightness {
			lightEmitter := types.LightEmitter{
				State: types.LightEmitterState{
					BrightnessInPercent: uint8(randBrightness),
				},
			}
			device := types.Device{
				Components: map[string]types.Component{
					"light emitting component 1": lightEmitter,
				},
			}
			keyX := types.ExternalDeviceID{Manufacturer: "foo", ID: "device X"}
			differ.Consider(map[types.ExternalDeviceID]types.Device{keyX: device})
			c.Assert(len(dest), Equals, 1, Commentf("iteration %v", i))
			obtained := <-dest

			expected := DeviceUpdated{
				ID:      keyX,
				NewState: device,
			}
			c.Assert(obtained, DeepEquals, expected, Commentf("iteration %v", i))
		}
		lastBrightness = randBrightness
	}
}

type diffDeviceTest struct {
	original, new                     types.Device
	expectedUpserted, expectedDeleted map[string]types.Component
	expectedDeviceChanged             bool
}

var diffDeviceTests = []diffDeviceTest{
	{},
	/*
		{
			original: types.Device{
				Components: map[string]types.Component{
					"light emitting component 1": types.LightEmitter{},
				},
			},
			new: types.Device{
				Components: map[string]types.Component{},
			},
			expectedDeleted: []string{"light emitting component 1"},
		},
	*/
}

func (s *TestSIFTLibSuite) TestDiffDevice(c *C) {
	for _, test := range diffDeviceTests {
		upserted, deleted, deviceChanged := DiffDevice(test.original, test.new)

		// Confirm that the 'upserted' maps are equal
		c.Assert(len(upserted), Equals, len(test.expectedUpserted))
		for key, comp := range upserted {
			expected, ok := test.expectedUpserted[key]
			c.Assert(ok, Equals, true, Commentf("obtained upserted map contained key %v, but the expected map did not", key))
			c.Assert(comp, DeepEquals, expected)
		}

		// Confirm that the 'deleted' maps are equal
		c.Assert(len(deleted), Equals, len(test.expectedDeleted))
		for key, comp := range deleted {
			expected, ok := test.expectedDeleted[key]
			c.Assert(ok, Equals, true, Commentf("obtained deleted map contained key %v, but the expected map did not", key))
			c.Assert(comp, DeepEquals, expected)
		}

		// Confirm that deviceChanged is as expected
		c.Assert(deviceChanged, Equals, test.expectedDeviceChanged)
	}
}
