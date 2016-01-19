package example

//import (
//	"gopkg.in/check.v1"
//	"net"
//	"net/http/httptest"
//	"net/url"
//	"github.com/upwrd/sift/network/ipv4"
//	"github.com/upwrd/sift/types"
//	"strconv"
//	"strings"
//	//"time"
//)
//
//func StartServerWithHTTPTest(c *check.C) (*Server, *httptest.Server) {
//	// Start a local server
//	server := NewServer(defaultPort)
//	c.Assert(server, check.NotNil)
//	go server.Serve()
//	router := server.Handlers("")
//	c.Assert(router, check.NotNil)
//	ts := httptest.NewServer(router)
//	c.Assert(ts, check.NotNil)
//	return server, ts
//}
//
///*
//func StartDriverAdaptingHTTP(c *C, d *driver, ts *httptest.Server) debuggableAdapter {
//	numAdaptersBeforeAdaptingHTTP := len(d.debugAdapters) // We'll check later that a new debugAdapter has been added
//
//	// Tell the driver to try adapting the server running on localhost
//	context := BuildIPv4ContextForServer(c, ts)
//	err := d.HandleIPv4(context)
//	c.Assert(err, IsNil)
//
//	// The status should come back as being handled
//	timeout := 1 * time.Second
//	select {
//	case status := <-context.Status:
//		c.Assert(status, Equals, ipv4.DriverStatusHandling)
//	// TODO: check status
//	case <-time.After(timeout):
//		log.Warn("After %v, the driver had not acknowledged that it was handling the context", timeout)
//		c.Fail()
//	}
//
//	c.Assert(len(d.debugAdapters), Equals, numAdaptersBeforeAdaptingHTTP + 1) // A single debug adapter should have been added
//	return d.debugAdapters[len(d.debugAdapters)-1] // return most-recently-added debugAdapter
//}
//*/
//
////func StartDriverAdaptingHTTP(c *C, d *driver, ts *httptest.Server) {
////	// Tell the driver to try adapting the server running on localhost
////	context := BuildIPv4ContextForServer(c, ts)
////	err := d.HandleIPv4(context)
////	c.Assert(err, IsNil)
////
////	// The status should come back as being handled
////	timeout := 1 * time.Second
////	select {
////	case status := <-context.Status:
////		c.Assert(status, Equals, ipv4.DriverStatusHandling)
////	// TODO: check status
////	case <-time.After(timeout):
////		Log.Error("Timed out waiting for driver to handle context", "timeout", timeout)
////		c.Fail()
////	}
////}
//
//func BuildIPv4ContextForServer(c *C, ts *httptest.Server) ipv4.ServiceContext {
//	tsURL, err := url.Parse(ts.URL)
//	c.Assert(err, IsNil)
//	parts := strings.Split(tsURL.Host, ":")
//	c.Assert(len(parts), Equals, 2)
//	ipStr, portStr := parts[0], parts[1]
//	ip := net.ParseIP(ipStr)
//	context := ipv4.BuildContext(ip)
//	if port, err := strconv.Atoi(portStr); err != nil {
//		Log.Warn("could not convert port string to int", "err", err)
//	} else {
//		if port < 0 || port > 65535 {
//			Log.Warn("got weird value for port, ignoring", "port", port)
//		}
//		context.SetPort(uint16(port))
//	}
//	return context
//}
//
//func ConvertDeviceFromExampleFormatToSIFT(d Device) types.Device {
//	return convertDevice(d)
//}
//
//func ConvertComponentFromExampleFormatToSIFT(c Component) (types.Component, error) {
//	return convertComponent(c)
//}
//
//// Debug stuff, for tests
//
//func (a *ipv4Adapter) ForceRefresh() {
//	Log.Debug("test framework forcing ipv4Adapter to refresh", "adapter", a)
//	a.debgForceRefresh <- struct{}{}
//}
//
///*
//type debuggableAdapter interface {
//	lib.Adapter
//	debugForceRefresh()
//}
//
//func (d *driver) DebugRefreshAllAdapters() {
//	for _, adapter := range d.debugAdapters {
//		adapter.debugForceRefresh()
//	}
//}
//*/
