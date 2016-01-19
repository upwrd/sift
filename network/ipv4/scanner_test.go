package ipv4

/*

import (
	. "gopkg.in/check.v1"
	"io"
	"net"
	"net/http"
	"time"
)

func (s *MySuite) TestScannerInterfaces(c *C) {
	scanner := NewScanner()
	c.Assert(len(scanner.interfaces), Not(Equals), 0)

	for ignoredInterface, _ := range ignoredIPv4Interfaces {
		_, ok := scanner.interfaces[ignoredInterface]
		c.Assert(ok, Equals, false)
	}
}

type testHandler struct{}

func (h testHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "hello, world!\n")
}

func (s *MySuite) TestNetworkBasics(c *C) {
	time.Sleep(10 * time.Second) // Give http listeners some time to stop

//		s9997 := &http.Server{
//			Addr:    ":9997",
//			Handler: testHandler{},
//		}
//		go s9997.ListenAndServe()
//		s10102 := &http.Server{
//			Addr:    ":10102",
//			Handler: testHandler{},
//		}
//		go s10102.ListenAndServe()

	listener, err := net.Listen("tcp", ":9997")
	c.Assert(err, IsNil)
	go http.Serve(listener, nil)
	listener2, err := net.Listen("tcp", ":10102")
	c.Assert(err, IsNil)
	go http.Serve(listener2, nil)
	<-time.After(1000 * time.Millisecond)

	for i := 0; i < 5; i++ {
		conn, err := net.DialTimeout("tcp", "localhost:9997", time.Second)
		c.Assert(err, IsNil)
		conn.Close()
	}
	for i := 0; i < 5; i++ {
		conn, err := net.DialTimeout("tcp", "localhost:10102", time.Second)
		c.Assert(err, IsNil)
		conn.Close()
	}
	for i := 0; i < 5; i++ {
		conn, err := net.DialTimeout("tcp", "localhost:9997", time.Second)
		c.Assert(err, IsNil)
		conn.Close()
		conn, err = net.DialTimeout("tcp", "localhost:10102", time.Second)
		c.Assert(err, IsNil)
		conn.Close()
	}

	// Close TCP listeners
	err = listener.Close()
	c.Assert(err, IsNil)
	err = listener2.Close()
	c.Assert(err, IsNil)
	time.Sleep(3 * time.Second) // Give http listeners some time to stop
}

func (s *MySuite) TestIPv4Scanner(c *C) {
	time.Sleep(10 * time.Second) // Give http listeners some time to stop
	scanner := NewScanner()
	scanner.refreshInterfaces()

	results := scanner.Scan()
	c.Assert(len(results), Equals, 0, Commentf("results: %+v", results))

	d1 := ServiceDescription{OpenPorts: []int{9997, 10102}}
	d1ID := scanner.AddDescription(d1)
	c.Assert(len(d1ID), Not(Equals), 0)

	results = scanner.Scan()
	c.Assert(len(results), Equals, 0)


//		s9997 := &http.Server{
//			Addr:    ":9997",
//			Handler: testHandler{},
//		}
//		go s9997.ListenAndServe()
//		s10102 := &http.Server{
//			Addr:    ":10102",
//			Handler: testHandler{},
//		}
//		go s10102.ListenAndServe()

	listener, err := net.Listen("tcp", ":9997")
	c.Assert(err, IsNil)
	go http.Serve(listener, nil)
	listener2, err := net.Listen("tcp", ":10102")
	c.Assert(err, IsNil)
	go http.Serve(listener2, nil)
	<-time.After(5 * time.Second)

	results = scanner.Scan()
	c.Assert(len(results), Equals, 1, Commentf("results: %+v", results))
	var foundIP string
	for ip, result := range results {
		c.Assert(len(result), Equals, 1)
		c.Assert(result[0], Equals, d1ID)
		foundIP = ip
	}

	results = scanner.Scan()
	c.Assert(len(results), Equals, 0) // Original IP should now be "locked"

	scanner.Unlock(net.ParseIP(foundIP))
	<-time.After(5 * time.Second)
	results = scanner.Scan()
	c.Assert(len(results), Equals, 1, Commentf("expected to find %s on the next scan after it was unlocked", foundIP))
	var foundIP2 string
	for ip, result := range results {
		c.Assert(len(result), Equals, 1)
		c.Assert(result[0], Equals, d1ID)
		foundIP2 = ip
	}
	c.Assert(foundIP2, Equals, foundIP)

	d2 := ServiceDescription{OpenPorts: []int{9997}}
	d2ID := scanner.AddDescription(d2)
	c.Assert(len(d2ID), Not(Equals), 0)

	scanner.Unlock(net.ParseIP(foundIP))
	results = scanner.Scan()
	c.Assert(len(results), Equals, 1)
	for _, result := range results {
		for _, id := range result {
			if id != d1ID && id != d2ID {
				log.Error("expected known id (either %v or %v), got %v", d1ID, d2ID, id)
				c.Fail()
			}
		}
	}

	// Close TCP listeners
	err = listener.Close()
	c.Assert(err, IsNil)
	err = listener2.Close()
	c.Assert(err, IsNil)
	time.Sleep(3 * time.Second) // Give http listeners some time to stop
}
*/
