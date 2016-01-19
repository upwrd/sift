package ipv4

import (
	. "gopkg.in/check.v1"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func TestSIFTIPv4(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})
