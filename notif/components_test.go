package notif_test

import (
	. "gopkg.in/check.v1"
	//"github.com/upwrd/sift"
	"github.com/upwrd/sift/auth"
	"github.com/upwrd/sift/notif"
	"github.com/upwrd/sift/types"
)

func (s *MySuite) TestComponents(c *C) {
	//sift.SetLogLevel("debug")
	a := auth.New()
	n := notif.New(a) // Create new notifier
	c.Assert(n, NotNil)
	token := a.Login()
	fooID := types.ComponentID{DeviceID: 1, Name: "foo"}
	barID := types.ComponentID{DeviceID: 1, Name: "bar"}

	// Create several listeners with different filters
	everything := n.Listen(token)
	allComponents := n.Listen(token, "components")
	allLights := n.Listen(token, notif.ComponentFilter{Type: types.ComponentTypeLightEmitter})
	allSpeakers := n.Listen(token, notif.ComponentFilter{Type: types.ComponentTypeSpeaker})
	fooOnly := n.Listen(token, notif.ComponentFilter{ID: fooID})
	fooUpdatesAndDeletes := n.Listen(token, notif.ComponentFilter{ID: fooID, Actions: notif.Update | notif.Delete})

	notif.Log.Debug("--test channels--", "everything", everything, "allComponents", allComponents, "allLights", allLights, "allSpeakers", allSpeakers)
	for _, val := range []<-chan interface{}{everything, allComponents, allLights, allSpeakers, fooOnly, fooUpdatesAndDeletes} {
		c.Assert(val, NotNil)
	}

	// Post a notification for a new light
	light := types.LightEmitter{}
	n.PostComponent(fooID, light, notif.Create)

	// Check that the appropriate notification channels got the notification (and that others didn't)
	c.Assert(len(everything), Equals, 1)
	c.Assert(len(allComponents), Equals, 1)
	c.Assert(len(allLights), Equals, 1)
	c.Assert(len(allSpeakers), Equals, 0)
	c.Assert(len(fooOnly), Equals, 1)
	c.Assert(len(fooUpdatesAndDeletes), Equals, 0)

	// ...and that they got the expected notification
	expected := notif.ComponentNotification{
		ID:        fooID,
		Action:    notif.Create,
		Component: light,
	}
	c.Assert(<-everything, DeepEquals, expected)
	c.Assert(<-allComponents, DeepEquals, expected)
	c.Assert(<-allLights, DeepEquals, expected)
	c.Assert(<-fooOnly, DeepEquals, expected)

	// Post a notification for an updated speaker
	speaker := types.Speaker{}
	n.PostComponent(barID, speaker, notif.Update)

	// Check that the appropriate notification channels got the notification (and that others didn't)
	c.Assert(len(everything), Equals, 1)
	c.Assert(len(allComponents), Equals, 1)
	c.Assert(len(allLights), Equals, 0)
	c.Assert(len(allSpeakers), Equals, 1)
	c.Assert(len(fooOnly), Equals, 0)
	c.Assert(len(fooUpdatesAndDeletes), Equals, 0)

	// ...and that they got the expected notification
	expected = notif.ComponentNotification{
		ID:        barID,
		Action:    notif.Update,
		Component: speaker,
	}
	c.Assert(<-everything, DeepEquals, expected)
	c.Assert(<-allComponents, DeepEquals, expected)
	c.Assert(<-allSpeakers, DeepEquals, expected)

	// Post a notification for an updated light
	light.State.BrightnessInPercent = 42
	n.PostComponent(fooID, light, notif.Update)

	// Check that the appropriate notification channels got the notification (and that others didn't)
	c.Assert(len(everything), Equals, 1)
	c.Assert(len(allComponents), Equals, 1)
	c.Assert(len(allLights), Equals, 1)
	c.Assert(len(allSpeakers), Equals, 0)
	c.Assert(len(fooOnly), Equals, 1)
	c.Assert(len(fooUpdatesAndDeletes), Equals, 1)

	// ...and that they got the expected notification
	expected = notif.ComponentNotification{
		ID:        fooID,
		Action:    notif.Update,
		Component: light,
	}
	c.Assert(<-everything, DeepEquals, expected)
	c.Assert(<-allComponents, DeepEquals, expected)
	c.Assert(<-allLights, DeepEquals, expected)
	c.Assert(<-fooOnly, DeepEquals, expected)
	c.Assert(<-fooUpdatesAndDeletes, DeepEquals, expected)
}
