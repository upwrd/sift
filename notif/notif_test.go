package notif_test

import (
	. "gopkg.in/check.v1"
	"github.com/upwrd/sift"
	"github.com/upwrd/sift/auth"
	"github.com/upwrd/sift/notif"
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
