package drivers_test

import (
	. "gopkg.in/check.v1"
	"github.com/upwrd/sift/drivers"
	"github.com/upwrd/sift/network/ipv4"
	"github.com/upwrd/sift/types"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func TestSIFTLib2(t *testing.T) { TestingT(t) }

type TestSIFTLibSuite struct{}

var _ = Suite(&TestSIFTLibSuite{})

type testAdapter struct {
	updateChan chan interface{}
	numEnacted int
	repeater   bool // if true, EnactIntent will put the count of numEnacted into updateChan
}

func (a *testAdapter) EnactIntent(k types.ExternalComponentID, i types.Intent) error {
	a.numEnacted++
	if a.repeater {
		a.updateChan <- a.numEnacted
	}
	return nil
}
func (a *testAdapter) UpdateChan() chan interface{} {
	return a.updateChan
}

// TestTestAdapter tests testAdapter (duh)
func (s *TestSIFTLibSuite) TestTestAdapter(c *C) {
	t := testAdapter{updateChan: make(chan interface{}, 10)}
	a := drivers.Adapter(&t)
	c.Assert(a, NotNil)

	// Test UpdateChan()
	c.Assert(a.UpdateChan(), NotNil)
	c.Assert(len(a.UpdateChan()), Equals, 0)
	t.updateChan <- "foobar"
	c.Assert(len(a.UpdateChan()), Equals, 1)
	c.Assert(<-a.UpdateChan(), Equals, "foobar")

	// Test EnactIntent
	c.Assert(t.numEnacted, Equals, 0)
	err := a.EnactIntent(types.ExternalComponentID{}, types.SetSpeakerIntent{})
	c.Assert(err, IsNil)
	c.Assert(t.numEnacted, Equals, 1)
}

//
// IPv4 Adapter Factory
//
type testIPv4AdapterFactory struct{}

func (t testIPv4AdapterFactory) HandleIPv4(ipv4.ServiceContext) drivers.Adapter {
	return &testAdapter{updateChan: make(chan interface{}, 10), repeater: true}
}
func (t testIPv4AdapterFactory) GetIPv4Description() ipv4.ServiceDescription {
	return ipv4.ServiceDescription{OpenPorts: []uint16{12345}}
}
func (t testIPv4AdapterFactory) Name() string { return "test_adapter_factory" }

func (s *TestSIFTLibSuite) TestIPv4AdapterFactory(c *C) {
	taf := testIPv4AdapterFactory{}
	t := drivers.IPv4DriverFactory(taf)
	c.Assert(t, NotNil)

	desc := t.GetIPv4Description()
	c.Assert(desc.OpenPorts, DeepEquals, []uint16{12345})

	a := t.HandleIPv4(ipv4.ServiceContext{})
	c.Assert(a, NotNil)
	c.Assert(len(a.UpdateChan()), Equals, 0)
	a.EnactIntent(types.ExternalComponentID{}, types.SetSpeakerIntent{}) // Noop intent, should repeat into updateChan
	c.Assert(len(a.UpdateChan()), Equals, 1)
}
