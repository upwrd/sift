package connectedbytcpnew

import (
	"encoding/xml"
	"fmt"
	"io"
	"github.com/upwrd/sift/types"
	"strconv"
)

const (
	componentManufacturer = "connected_by_tcp"
)

// XMLGWRCmds captures the data from a similarly-named Connected by TCP API structure
type XMLGWRCmds struct {
	XMLName  xml.Name    `xml:"gwrcmds"`
	Commands []XMLGWRCmd `xml:"gwrcmd"`
}

// XMLGWRCmd captures the data from a similarly-named Connected by TCP API structure
type XMLGWRCmd struct {
	XMLName xml.Name `xml:"gwrcmd"`
	Command XMLGCmd  `xml:"gcmd"`
	Data    XMLGData `xml:"gdata"`
}

// XMLGCmd captures the data from a similarly-named Connected by TCP API structure
type XMLGCmd struct {
	XMLName xml.Name `xml:"gcmd"`
	Value   string   `xml:",chardata"`
}

// XMLGData captures the data from a similarly-named Connected by TCP API structure
type XMLGData struct {
	XMLName xml.Name `xml:"gdata"`
	Gips    []XMLGip `xml:"gip"`
}

// XMLGip captures the data from a similarly-named Connected by TCP API structure
type XMLGip struct {
	XMLName xml.Name  `xml:"gip"`
	Rooms   []XMLRoom `xml:"room"`
}

// XMLRoom captures the data from a similarly-named Connected by TCP API structure
type XMLRoom struct {
	XMLName xml.Name    `xml:"room"`
	Devices []XMLDevice `xml:"device"`
}

// XMLDevice captures the data from a similarly-named Connected by TCP API structure
type XMLDevice struct {
	XMLName xml.Name         `xml:"device"`
	DID     XMLDID           `xml:"did"`
	Name    XMLDeviceName    `xml:"name"`
	Offline XMLDeviceOffline `xml:"offline"`
	State   XMLDeviceState   `xml:"state"`
	Level   XMLDeviceLevel   `xml:"level"`
}

// XMLDID captures the data from a similarly-named Connected by TCP API structure
type XMLDID struct {
	XMLName xml.Name `xml:"did"`
	Value   string   `xml:",chardata"`
}

// XMLDeviceName captures the data from a similarly-named Connected by TCP API structure
type XMLDeviceName struct {
	XMLName xml.Name `xml:"name"`
	Value   string   `xml:",chardata"`
}

// XMLDeviceOffline captures the data from a similarly-named Connected by TCP API structure
type XMLDeviceOffline struct {
	XMLName xml.Name `xml:"offline"`
	Value   string   `xml:",chardata"`
}

// XMLDeviceState captures the data from a similarly-named Connected by TCP API structure
type XMLDeviceState struct {
	XMLName xml.Name `xml:"state"`
	Value   string   `xml:",chardata"`
}

// XMLDeviceLevel captures the data from a similarly-named Connected by TCP API structure
type XMLDeviceLevel struct {
	XMLName xml.Name `xml:"level"`
	Value   string   `xml:",chardata"`
}

// XMLLoginGip captures the data from a similarly-named Connected by TCP API structure
type XMLLoginGip struct {
	XMLName xml.Name   `xml:"gip"`
	Version XMLVersion `xml:"version"`
	RC      XMLRC      `xml:"rc"`
	Token   XMLToken   `xml:"token"`
	Rooms   []XMLRoom  `xml:"room"`
}

// XMLVersion captures the data from a similarly-named Connected by TCP API structure
type XMLVersion struct {
	XMLName xml.Name `xml:"version"`
	Value   string   `xml:",chardata"`
}

// XMLRC captures the data from a similarly-named Connected by TCP API structure
type XMLRC struct {
	XMLName xml.Name `xml:"rc"`
	Value   string   `xml:",chardata"`
}

// XMLToken captures the data from a similarly-named Connected by TCP API structure
type XMLToken struct {
	XMLName xml.Name `xml:"token"`
	Value   string   `xml:",chardata"`
}

// ReadDevice marshals valid XML describing a Device from the Connected by TCP API into an XMLDevice struct
func ReadDevice(xmlStr []byte) (*XMLDevice, error) {
	var xmlDevice XMLDevice
	if err := xml.Unmarshal(xmlStr, &xmlDevice); err != nil {
		return nil, err
	}

	return &xmlDevice, nil
}

// ReadRoom marshals valid XML describing a Room from the Connected by
// TCP API into a collection of XMLDevice structs
func ReadRoom(xmlStr []byte) ([]XMLDevice, error) {
	var xmlRoom XMLRoom
	if err := xml.Unmarshal(xmlStr, &xmlRoom); err != nil {
		return nil, err
	}

	return xmlRoom.Devices, nil
}

// ReadGwrcmds marshals valid XML describing Gwrcmds from the Connected by
// TCP API into a collection of XMLGWRCmd structs
func ReadGwrcmds(xmlStr []byte) ([]XMLGWRCmd, error) {
	var xmlGWRCmds XMLGWRCmds
	if err := xml.Unmarshal(xmlStr, &xmlGWRCmds); err != nil {
		return nil, err
	}

	return xmlGWRCmds.Commands, nil
}

// DecodeGwrcmds marshals valid XML describing Gwrcmds from the Connected by
// TCP API into a collection of XMLGWRCmd structs
func DecodeGwrcmds(r io.Reader) ([]XMLGWRCmd, error) {
	var xmlGWRCmds XMLGWRCmds
	dec := xml.NewDecoder(r)

	err := dec.Decode(&xmlGWRCmds)
	return xmlGWRCmds.Commands, err
}

// ReadGwrLogin marshals valid XML describing GwrLogins from the Connected by
// TCP API into a collection of XMLLoginGip structs
func ReadGwrLogin(xmlStr []byte) (XMLLoginGip, error) {
	var loginGip XMLLoginGip
	err := xml.Unmarshal(xmlStr, &loginGip)
	return loginGip, err
}

// XMLDeviceToSIFTDevice converts a Connected By TCP XMLDevice into a SIFT
// types.Device
func XMLDeviceToSIFTDevice(device XMLDevice) (types.Device, error) {
	var outputInPercent int

	if device.Level.Value != "" {
		// TCP's API includes a state flag where 0 indicates offline, 1 indicates online
		// If state == 0, the output in percent should be 0, regardless of the given output value
		if device.State.Value != "" {
			state, err := strconv.Atoi(device.State.Value)
			if err != nil {
				return types.Device{}, fmt.Errorf("could not convert XML State string %v to int: %v", device.Level.Value, err)
			}
			if state == 0 {
				outputInPercent = 0
			} else {
				level, err := strconv.Atoi(device.Level.Value)
				if err != nil {
					return types.Device{}, fmt.Errorf("could not convert XML Level string %v to int: %v", device.Level.Value, err)
				}
				switch level {
				case 0:
					outputInPercent = 1 // TCP sets level==0 when the bulb is still technically on
				default:
					outputInPercent = level
				}
			}
		}
	}

	// If device.Offline is not empty, this is offline
	var isOnline bool
	if device.Offline.Value == "" {
		isOnline = true
	}

	return types.Device{
		Name:     "TCP bulb " + device.Name.Value,
		IsOnline: isOnline,
		Components: map[string]types.Component{
			device.DID.Value: types.LightEmitter{
				BaseComponent: types.BaseComponent{
					Make:  "connected_by_tcp",
					Model: "bulb",
				},
				State: types.LightEmitterState{
					BrightnessInPercent: uint8(outputInPercent),
				},
			},
		},
	}, nil
}
