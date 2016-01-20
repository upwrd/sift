package connectedbytcpnew

import (
	"bytes"
	uuidlib "code.google.com/p/go-uuid/uuid"
	"crypto/tls"
	"fmt"
	"github.com/upwrd/sift/adapter"
	"github.com/upwrd/sift/lib"
	"github.com/upwrd/sift/logging"
	"github.com/upwrd/sift/network/ipv4"
	"github.com/upwrd/sift/types"
	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Log is used to log messages for the connectedbytcp package. Logs are
// disabled by default; use sift/logging.SetLevel() to set log levels for all
// packages, or Log.SetHandler() to set a custom handler for this package (see:
// https://godoc.org/gopkg.in/inconshreveable/log15.v2)
var Log = logging.Log.New("pkg", "drivers/connectedbytcp")

const (
	openPorts             = 443
	timeBetweenHeartbeats = 5 * time.Second
	timeBetweenPolls      = 10 * time.Second

	numLoginRetries               = 3
	credentialsKeyGatewayToken    = "tcpGatewayToken"
	credentialsKeyGatewayUUID     = "tcpGatewayUUID"
	tcpGatewayCredentialsTokenKey = "cbtcpGateway"
)

// An AdapterFactory creates adapters
type AdapterFactory struct{}

// NewFactory properly instantiates a new AdapterFactory
func NewFactory() *AdapterFactory { return &AdapterFactory{} }

// HandleIPv4 spawns a new Adapter to handle a context
func (f *AdapterFactory) HandleIPv4(context *ipv4.ServiceContext) adapter.Adapter {
	if context == nil {
		return nil
	}
	return newIPv4Adapter(context)
}

// GetIPv4Description returns a description of the example IPv4 service that
// can be used to identify example services on a network
func (f *AdapterFactory) GetIPv4Description() ipv4.ServiceDescription {
	return ipv4.ServiceDescription{OpenPorts: []uint16{openPorts}}
}

// Name returns the name of this adapter factory, "Connected By TCP"
func (f *AdapterFactory) Name() string { return "Connected by TCP" }

type ipv4Adapter struct {
	updateChan chan interface{}
	context    *ipv4.ServiceContext
	differ     lib.SetOutputBasedDeviceDiffer
	stop       chan struct{}
	log        log.Logger
}

func newIPv4Adapter(context *ipv4.ServiceContext) *ipv4Adapter {
	log := Log.New("obj", "Connected By TCP IPv4 Adapter", "id", logext.RandId(8), "adapting", context.IP.String())
	adapter := &ipv4Adapter{
		updateChan: make(chan interface{}, 100),
		context:    context,
		differ:     lib.NewAllAtOnceDiffer(),
		stop:       make(chan struct{}),
		log:        log,
	}
	if err := adapter.differ.SetOutput(adapter.updateChan); err != nil {
		panic(fmt.Sprintf("newAdapter() could not set output: %v", err))
	}
	go adapter.Serve()
	return adapter
}

// Serve begins adapting the example service specified by the adapter's
// context. As updates within the service are found, they will be sent to the
// update channel provided by UpdateChan(). While the adapter is serving,
// heartbeat messages will be sent to the adapter's context's status channel.
func (a *ipv4Adapter) Serve() {
	// Check if the ipv4 context that we were given represents a Connected By TCP service
	if !a.isConnectedByTCPService() {
		a.log.Info("service is not a Connected By TCP service", "ip", a.context.IP.String())
		a.context.SendStatus(ipv4.AdapterStatusIncorrectService)
		return
	}

	if a.differ == nil {
		a.log.Warn("Connected By TCP IPv4 Adapter was improperly instantiated!")
		a.context.SendStatus(ipv4.AdapterStatusError)
		return
	}

	// Send heartbeats to the caller as long as this service is being handled.
	stopHeartbeating := make(chan struct{})
	defer func() {
		stopHeartbeating <- struct{}{}
	}()
	go func() {
		heartbeat := time.NewTimer(0)
		for {
			select {
			case <-stopHeartbeating:
				return
			case <-heartbeat.C:
				// Try to send a heartbeat status
				if err := a.context.SendStatus(ipv4.AdapterStatusHandling); err != nil {
					return // Context must have been killed, stop heartbeating
				}
				heartbeat.Reset(timeBetweenHeartbeats)
			}
		}
	}()

	// Periodically gather states from the server
	timer := time.NewTimer(timeBetweenPolls)
	for {
		timer.Reset(timeBetweenPolls)
		select {
		case <-timer.C: // if the timer signal is recieved, continue
		case <-a.stop:
			return // if the stop signal is received, exit from the function
		}

		devices, err := getDevicesFromServer(a.context)
		if err != nil {
			a.log.Warn("error getting devices from server", "err", err)
			a.context.SendStatus(ipv4.AdapterStatusError)
			return
		}

		a.log.Debug("driver got devices from server", "devices", devices)
		// Just got a batch of devices - send them to the differ to see if any devices were changed or removed
		a.differ.Consider(devices)
	}
}

// Stop stops the adapter
func (a *ipv4Adapter) Stop() { a.stop <- struct{}{} }

// UpdateChan returns a channel which will be populated with updates from the
// adapter
func (a *ipv4Adapter) UpdateChan() chan interface{} { return a.updateChan }

func (a *ipv4Adapter) isConnectedByTCPService() bool {
	// log in
	_, err := loginWRetry(a.context, numLoginRetries) // ignore output, only care if its successful
	if err != nil {
		a.log.Warn("endpoint is NOT a Connected By TCP service, because login was unsuccessful", "err", err, "service_ip", a.context.IP)
		return false
	}

	client := getCertificateIgnoringClient()
	endpointURL := "https://" + a.context.IP.String() + "/gwr/gop.php"
	resp, err := client.Get(endpointURL)
	if err != nil {
		a.log.Warn("endpoint is NOT a Connected By TCP service, because of error during http.Get", "err", err, "url_attempted", endpointURL)
		return false
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		a.log.Warn("service is NOT a Connected By TCP service, because the content of the endpoint path was not readable", "err", err, "url_attempted", endpointURL)
		return false
	}

	expectedContent := "Missing input"
	bodyStr := string(body)
	if !strings.Contains(bodyStr, expectedContent) {
		a.log.Warn("service is NOT a Connected By TCP service, because the expected content did not match", "expected", expectedContent, "got", bodyStr, "url_attempted", endpointURL)
		return false
	}

	a.log.Debug("found a ConnectedByTCP Gateway", "url", endpointURL)
	return true
}

func getDevicesFromServer(context *ipv4.ServiceContext) (map[types.ExternalDeviceID]types.Device, error) {
	// Request XML from the ConnectedByTCP API running on the gateway at gatewayIP
	gatewayXML, err := getXMLFromGateway(context)
	if err != nil {
		return nil, fmt.Errorf("could not get device data from server: %v", err)
	}

	// Parse the XML into structs
	gwrCommands, err := ReadGwrcmds([]byte(gatewayXML))
	if err != nil {
		//TODO: What if this XML describes an authentication error?
		// We should indicate to caller, so they can reauth and try again...
		Log.Debug("could not parse ConnectedByTCP XML into internal structs", "err", err, "resp_body", gatewayXML)
		return nil, fmt.Errorf("could not parse ConnectedByTCP XML into internal structs: %v", err)
	}

	// Walk through the internal structs and pull out any Devices that were described
	devices := make(map[types.ExternalDeviceID]types.Device)
	for _, gwrCommand := range gwrCommands {
		if gwrCommand.Command.Value == "RoomGetCarousel" {
			for _, gip := range gwrCommand.Data.Gips {
				for _, room := range gip.Rooms {
					for _, device := range room.Devices {
						deviceConvertedToSift, err := XMLDeviceToSIFTDevice(device)
						if err != nil {
							return nil, fmt.Errorf("could not convert XML device to SIFT format: %v", err)
						}
						key := types.ExternalDeviceID{
							Manufacturer: "TCP",
							ID:           device.DID.Value,
						}
						devices[key] = deviceConvertedToSift
					}
				}
			}
		}
	}
	return devices, nil
}

func loginWRetry(context *ipv4.ServiceContext, numRetries int) (string, error) {
	for i := 0; i < numRetries; i++ {
		token, err := loginContext(context)
		if err != nil {
			Log.Debug("login to Connected by TCP hub failed", "ip", context.IP.String(), "attempt", i+1, "max_attempts", numRetries, "err", err)
			if i == numRetries-1 {
				return "", fmt.Errorf("could not log in to Connected By TCP hub: %v (tries: %v)\n", err, i+1)
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}
		// If we've reached this spot, login was successful and token should be valid
		return token, nil
	}
	return "", nil // Should never reach this spot
}

// loginContext gets valid credentials from the ConnectedByTCP Gateway and
// stores them in the context
func loginContext(context *ipv4.ServiceContext) (string, error) {
	uuid, token := getUUIDFromContext(context), getLoginTokenFromContext(context)
	if uuid != "" && token != "" {
		Log.Debug("using cached credentials", "uuid", uuid, "token", token)
		return token, nil // already logged in
	}

	// If no UUID was retrieved, create a new one to log in with
	if uuid == "" {
		uuid = uuidlib.New()
	}

	// format the login request
	values := make(url.Values)
	values.Set("cmd", "GWRLogin")
	values.Set("fmt", "xml")
	command := "<gip><version>1</version><email>" + uuid + "</email><password>" + uuid + "</password></gip>"
	values.Set("data", command)

	// post request, get response
	client := getCertificateIgnoringClient() // TODO: replace with context client
	url := "https://" + context.IP.String() + "/gwr/gop.php"
	resp, err := client.PostForm(url, values)
	if err != nil {
		Log.Debug("failed to post login form", "err", err, "url", url, "uuid", uuid, "form_values_submitted", values.Encode())
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log.Debug("could not read response body", "err", err, "body", resp.Body, "url", url, "uuid", uuid, "form_values_submitted", values.Encode())
		return "", err
	}

	// parse the request into an xmlLogin object
	xmlLogin, err := ReadGwrLogin(body)
	if err != nil {
		return "", err
	}

	// TODO: Does a response of <gip><version>1</version><rc>404</rc></gip>
	// indicate that the user needs to press the login button?
	if xmlLogin.RC.Value != "200" || xmlLogin.Token.Value == "" {
		Log.Debug("could not parse login values from response", "err", err, "body", string(body), "url", url, "uuid", uuid, "form_values_submitted", values.Encode())
		authErr := authFailedError{
			When:      time.Now(),
			What:      "tried to login to ConnectedByTCP API at " + context.IP.String() + ", but received unexpected response: " + string(body),
			WhatDoIDo: "press the connect button on top of the ConnectedByTCP Gateway",
		}
		return "", authErr
	}
	token = xmlLogin.Token.Value
	// Store the UUID and token
	if err := saveUUIDAndTokenToContext(context, uuid, token); err != nil {
		return "", err
	}
	Log.Debug("logged in to ConnectedByTCP service", "uuid", uuid, "token", token)
	return token, nil // happy path
}

type authFailedError struct {
	When      time.Time
	What      string // A techy description of what went wrong
	WhatDoIDo string // If the user needs to take action, let them know how
}

func (e authFailedError) Error() string {
	return e.What
}

// getXMLFromGateway queries a ConnectedByTCP Gateway API for XML representing
// the current state of connected devices
func getXMLFromGateway(context *ipv4.ServiceContext) (string, error) {
	token, err := loginWRetry(context, numLoginRetries)
	if err != nil {
		return "", fmt.Errorf("could not log in to Connected By TCP hub")
	}
	// These values form a query to the ConnectedByTCP API
	values := make(url.Values)
	values.Set("cmd", "GWRBatch")
	values.Set("fmt", "xml")
	values.Set("data", "<gwrcmds><gwrcmd><gcmd>RoomGetCarousel</gcmd><gdata><gip><version>1</version><token>"+token+"</token><fields>name,image,imageurl,control,power,product,class,realtype,status</fields></gip></gdata></gwrcmd></gwrcmds>")

	// BUG(donald): The ConnectedByTCP Gateway produces a x509 certificate
	// which is not recognized by the default HTTP client.  This adapter
	// currently ignores the certificate check (!!!)
	client := getCertificateIgnoringClient()
	response, err := client.PostForm("https://"+context.IP.String()+"/gwr/gop.php", values)
	if err != nil {
		return "", err
	}
	defer func() { response.Close = true }()

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	return buf.String(), nil
}

// getCertificateIgnoringClient gets an http client which ignores certificates
// TODO: record certificate appropriately
func getCertificateIgnoringClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{Transport: tr}
}

// EnactIntent will attempt to satisfy the provided intent by sending network
// messages to the Devices specified by target.
func (a *ipv4Adapter) EnactIntent(target types.ExternalComponentID, intent types.Intent) error {
	switch typed := intent.(type) {
	default:
		return fmt.Errorf("unhandled intent type %T", intent)
	case types.SetLightEmitterIntent:
		return setLight(a.context, target, typed)
	}
}

// setLight sends a command to the Connected By TCP hub to change a light
func setLight(context *ipv4.ServiceContext, target types.ExternalComponentID, intent types.SetLightEmitterIntent) error {
	token, err := loginWRetry(context, numLoginRetries)
	if err != nil {
		return fmt.Errorf("could not log in to Connected By TCP hub")
	}

	// Get the destination path
	apiAddr := "https://" + context.IP.String() + "/gwr/gop.php"

	// Convert some values into the appropriate string format
	var isOn string // "0" == off, "1" == on
	if intent.BrightnessInPercent == 0 {
		isOn = "0"
	} else {
		isOn = "1"
	}
	brightnessStr := strconv.Itoa(int(intent.BrightnessInPercent))

	// Build the post values to match Connected by TCP API specs
	command := "<gwrcmds>"
	values := make(url.Values)
	values.Set("cmd", "GWRBatch")
	values.Set("fmt", "xml")
	command += "<gwrcmd><gcmd>DeviceSendCommand</gcmd><gdata><gip><version>1</version><token>" + token + "</token><did>" + target.Name + "</did><value>" + brightnessStr + "</value><type>level</type></gip></gdata></gwrcmd><gwrcmd><gcmd>DeviceSendCommand</gcmd><gdata><gip><version>1</version><token>" + token + "</token><did>" + target.Name + "</did><value>" + isOn + "</value></gip></gdata></gwrcmd>"
	command += "</gwrcmds>"
	values.Set("data", command)

	client := getCertificateIgnoringClient()
	Log.Debug("Posting form to change light", "addr", apiAddr, "values", values)
	_, err = client.PostForm(apiAddr, values)
	return err
}

func uuidKey(unique string) string  { return credentialsKeyGatewayUUID + ":" + unique }
func tokenKey(unique string) string { return credentialsKeyGatewayToken + ":" + unique }

func saveUUIDToContext(context *ipv4.ServiceContext, uuid string) error {
	// unique is a string identifier which should be locally unique for services
	// TODO: replace with better unique identifier than IP, if one is available
	unique := context.IP.String()
	key := uuidKey(unique)
	return context.StoreData(key, uuid)
}

func getUUIDFromContext(context *ipv4.ServiceContext) string {
	// unique is a string identifier which should be locally unique for services
	// TODO: replace with better unique identifier than IP, if one is available
	unique := context.IP.String()
	key := uuidKey(unique)
	uuid, err := context.GetData(key) // get the UUID
	if err != nil {
		Log.Warn("could not get UUID from context", "err", err, "uuid_key", key, "context", context)
		return ""
	}
	return uuid
}

func saveLoginTokenToContext(context *ipv4.ServiceContext, token string) error {
	// unique is a string identifier which should be locally unique for services
	// TODO: replace with better unique identifier than IP, if one is available
	unique := context.IP.String()
	key := tokenKey(unique)
	return context.StoreData(key, token)
}

func getLoginTokenFromContext(context *ipv4.ServiceContext) string {
	// unique is a string identifier which should be locally unique for services
	// TODO: replace with better unique identifier than IP, if one is available
	unique := context.IP.String()
	key := tokenKey(unique)
	token, err := context.GetData(key)
	if err != nil {
		Log.Warn("could not get token from context", "err", err, "token_key", key, "context", context)
		return ""
	}
	return token
}

func saveUUIDAndTokenToContext(context *ipv4.ServiceContext, uuid, token string) error {
	if err := saveUUIDToContext(context, uuid); err != nil {
		return fmt.Errorf("could not store uuid to context: %v", err)
	}
	if err := saveLoginTokenToContext(context, token); err != nil {
		return fmt.Errorf("could not store login token to context: %v", err)
	}
	return nil
}
