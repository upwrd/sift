package connectedbytcpnew

import (
	. "gopkg.in/check.v1"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func TestXML(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

var testDevice = `<device>
<did>216409378981665377</did>
<known>1</known>
<lock>0</lock>
<state>0</state>
<offline>1</offline>
<node>0</node>
<port>0</port>
<nodetype>16386</nodetype>
<name>track3</name>
<colorid>1</colorid>
<other>
<rcgroup/>
<manufacturer>TCP</manufacturer>
</other>
</device>
`

var testRoom = `
		<room>
		  <device>
            <did>216409378981665377</did>
            <known>1</known>
            <lock>0</lock>
            <state>0</state>
            <offline>1</offline>
            <node>0</node>
            <port>0</port>
            <nodetype>16386</nodetype>
            <name>track3</name>
            <colorid>1</colorid>
            <other>
              <rcgroup/>
              <manufacturer>TCP</manufacturer>
            </other>
          </device>
          <device>
            <did>216409378981741180</did>
            <known>1</known>
            <lock>0</lock>
            <state>0</state>
            <offline>1</offline>
            <node>0</node>
            <port>0</port>
            <nodetype>16386</nodetype>
            <name>track1</name>
            <colorid>1</colorid>
            <other>
              <rcgroup/>
              <manufacturer>TCP</manufacturer>
            </other>
          </device>
        </room>
        `

var testGwrcmds = `
<gwrcmds>
    <gwrcmd>
        <gcmd>RoomGetCarousel</gcmd>
        <gdata>
            <gip>
                <version>1</version>
                <rc>200</rc>
                <room>
                    <rid>1</rid>
                    <name>entry</name>
                    <desc/>
                    <known>1</known>
                    <type>0</type>
                    <color>00bd1f</color>
                    <colorid>1</colorid>
                    <img>img/room/green.png</img>
                    <power>0</power>
                    <poweravg>0</poweravg>
                    <energy>0</energy>
                    <device>
                        <did>216409378981665377</did>
                        <known>1</known>
                        <lock>0</lock>
                        <state>0</state>
                        <offline>1</offline>
                        <node>0</node>
                        <port>0</port>
                        <nodetype>16386</nodetype>
                        <name>track3</name>
                        <colorid>1</colorid>
                        <other>
                            <rcgroup/>
                            <manufacturer>TCP</manufacturer>
                        </other>
                    </device>
                    <device>
                        <did>216409378981741180</did>
                        <known>1</known>
                        <lock>0</lock>
                        <state>0</state>
                        <offline>1</offline>
                        <node>0</node>
                        <port>0</port>
                        <nodetype>16386</nodetype>
                        <name>track1</name>
                        <colorid>1</colorid>
                        <other>
                            <rcgroup/>
                            <manufacturer>TCP</manufacturer>
                        </other>
                    </device>
                    <device>
                        <did>216409378981745678</did>
                        <known>1</known>
                        <lock>0</lock>
                        <state>0</state>
                        <offline>1</offline>
                        <node>0</node>
                        <port>0</port>
                        <nodetype>16386</nodetype>
                        <name>track6</name>
                        <colorid>1</colorid>
                        <other>
                            <rcgroup/>
                            <manufacturer>TCP</manufacturer>
                        </other>
                    </device>
                    <device>
                        <did>216409378981755754</did>
                        <known>1</known>
                        <lock>0</lock>
                        <state>1</state>
                        <level>100</level>
                        <node>47</node>
                        <port>0</port>
                        <nodetype>16386</nodetype>
                        <name>librarian</name>
                        <desc>LED</desc>
                        <colorid>1</colorid>
                        <other>
                            <rcgroup/>
                            <manufacturer>TCP</manufacturer>
                            <capability>productinfo,identify,meter_power,switch_binary,switch_multilevel</capability>
                            <bulbpower>11</bulbpower>
                        </other>
                    </device>
                    <device>
                        <did>216409378981979856</did>
                        <known>1</known>
                        <lock>0</lock>
                        <state>0</state>
                        <offline>1</offline>
                        <node>0</node>
                        <port>0</port>
                        <nodetype>16386</nodetype>
                        <name>track5</name>
                        <colorid>1</colorid>
                        <other>
                            <rcgroup/>
                            <manufacturer>TCP</manufacturer>
                        </other>
                    </device>
                    <device>
                        <did>216409378982064569</did>
                        <known>1</known>
                        <lock>0</lock>
                        <state>0</state>
                        <offline>1</offline>
                        <node>0</node>
                        <port>0</port>
                        <nodetype>16386</nodetype>
                        <name>track2</name>
                        <colorid>1</colorid>
                        <other>
                            <rcgroup/>
                            <manufacturer>TCP</manufacturer>
                        </other>
                    </device>
                    <device>
                        <did>216409378982232743</did>
                        <known>1</known>
                        <lock>0</lock>
                        <state>0</state>
                        <offline>1</offline>
                        <node>0</node>
                        <port>0</port>
                        <nodetype>16386</nodetype>
                        <name>track4</name>
                        <colorid>1</colorid>
                        <other>
                            <rcgroup/>
                            <manufacturer>TCP</manufacturer>
                        </other>
                    </device>
                </room>
                <room>
                    <rid>2</rid>
                    <name>living</name>
                    <desc/>
                    <known>1</known>
                    <type>0</type>
                    <color>004fd9</color>
                    <colorid>2</colorid>
                    <img>img/room/blue.png</img>
                    <power>0</power>
                    <poweravg>0</poweravg>
                    <energy>0</energy>
                    <device>
                        <did>216409378981234199</did>
                        <known>1</known>
                        <lock>0</lock>
                        <state>1</state>
                        <level>100</level>
                        <node>47</node>
                        <port>0</port>
                        <nodetype>16386</nodetype>
                        <name>standing2</name>
                        <desc>LED</desc>
                        <colorid>2</colorid>
                        <other>
                            <rcgroup/>
                            <manufacturer>TCP</manufacturer>
                            <capability>productinfo,identify,meter_power,switch_binary,switch_multilevel</capability>
                            <bulbpower>11</bulbpower>
                        </other>
                    </device>
                    <device>
                        <did>216409378981634751</did>
                        <known>1</known>
                        <lock>0</lock>
                        <state>0</state>
                        <offline>1</offline>
                        <node>0</node>
                        <port>0</port>
                        <nodetype>16386</nodetype>
                        <name>living1</name>
                        <colorid>2</colorid>
                        <other>
                            <rcgroup/>
                            <manufacturer>TCP</manufacturer>
                        </other>
                    </device>
                    <device>
                        <did>216409378981955226</did>
                        <known>1</known>
                        <lock>0</lock>
                        <state>0</state>
                        <offline>1</offline>
                        <node>0</node>
                        <port>0</port>
                        <nodetype>16386</nodetype>
                        <name>standing1</name>
                        <colorid>2</colorid>
                        <other>
                            <rcgroup/>
                            <manufacturer>TCP</manufacturer>
                        </other>
                    </device>
                </room>
            </gip>
        </gdata>
    </gwrcmd>
</gwrcmds>
`

func (s *MySuite) TestDeviceConversion(c *C) {
	_, err := ReadDevice([]byte(testDevice))
	c.Assert(err, IsNil)
	//fmt.Printf("GOT DEV: %+v\n", dev)
}

func (s *MySuite) TestDevicesConversion(c *C) {
	_, err := ReadRoom([]byte(testRoom))
	c.Assert(err, IsNil)
	//fmt.Printf("GOT Room: %+v\n", devs)
}

func (s *MySuite) TestGWRCommandsConversion(c *C) {
	_, err := ReadGwrcmds([]byte(testGwrcmds))
	c.Assert(err, IsNil)
	// TODO: confirm that ReadGwrcmds is producing the expected output!
}
