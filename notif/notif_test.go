package notif_test

import (
	. "gopkg.in/check.v1"
	"sift"
	"sift/auth"
	"sift/notif"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func TestNotif(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestNotifier(c *C) {
	sift.SetLogLevel("error")
	n := notif.New(auth.New())
	c.Check(n, NotNil)
}
