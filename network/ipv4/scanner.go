package ipv4

import (
	"fmt"
	"github.com/pborman/uuid"
	"github.com/thejerf/suture"
	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	numIPv4Checkers = 100
)

var (
	ignoredIPv4Interfaces = map[string]struct{}{
		// DARWIN
		"lo0":  struct{}{}, // loopback
		"tun0": struct{}{}, // tunnel (VPN)
		// TODO: Windows ignored?
	}
)

// An IScanner searches the IPv4 network for services matching given
// descriptions. Once services are found, they are locked and returned to the
// caller. It is up to the caller to unlock IPs (via Unlock()) if they are no
// longer in use.
type IScanner interface {
	// Add a description to search for; return the ID used if a match is returned
	AddDescription(desc ServiceDescription) string

	// Scan the IPv4 network for services matching given descriptions.
	// Returns a map of IPs (string encoded) pointing to the IDs of those descriptions which matched
	Scan() map[string][]string

	// By default, after an IP is found with Scan it is ignored in future searches.
	// Unlock instructs the scanner to include responses for that IP address in future scans.
	Unlock(ip net.IP)
}

// A ServiceFoundNotification indicates that a service was found at ip IP which
// matched all of MatchingDescriptionIDs.
type ServiceFoundNotification struct {
	IP                     net.IP
	MatchingDescriptionIDs []string
}

// Scanner implements IScanner. It searches the IPv4 network for services
// matching given descriptions. Once services are found, they are locked and
// returned to the caller. It is up to the caller to unlock IPs (via Unlock())
// if they are no longer in use.
type Scanner struct {
	interfaces map[string]net.Interface
	ilock      sync.RWMutex // protects interfaces

	descriptionsByID map[string]ServiceDescription
	dlock            sync.RWMutex // protects descriptionsByID

	activeServicesByIP map[string]struct{}
	slock              *sync.RWMutex // protects activeServicesByIP

	log log.Logger
}

// NewScanner properly instantiates a Scanner.
func NewScanner() *Scanner {
	s := &Scanner{
		interfaces: make(map[string]net.Interface),
		ilock:      sync.RWMutex{},

		descriptionsByID: make(map[string]ServiceDescription),
		dlock:            sync.RWMutex{},

		activeServicesByIP: make(map[string]struct{}),
		slock:              &sync.RWMutex{},

		log: Log.New("obj", "ipv4.scanner", "id", logext.RandId(8)),
	}

	err := s.refreshInterfaces()
	if err != nil {
		panic("ipv4.NewScanner(): scanner could not refresh interfaces: " + err.Error())
	}

	return s
}

// AddDescription adds a ServiceDescription to the Scanner. On following scans,
// the Scanner will find services which match the description.
func (s *Scanner) AddDescription(desc ServiceDescription) string {
	s.dlock.Lock()
	defer s.dlock.Unlock()
	id := uuid.New()
	s.descriptionsByID[id] = desc
	return id
}

// Scan the IPv4 network for services matching given descriptions.
// Returns a map of IPs (string encoded) pointing to the IDs of those descriptions which matched
func (s *Scanner) Scan() map[string][]string {
	if len(s.descriptionsByID) == 0 {
		s.log.Debug("scanner has no descriptions, ignoring scan")
		return map[string][]string{}
	}

	s.log.Debug("ip4v scanner beginning scan", "interfaces", s.interfaces, "target_descriptions", s.descriptionsByID)
	foundServices := make(map[string][]string)
	flock := sync.Mutex{} // protects foundServices
	var wg sync.WaitGroup
	var numChecked, numFound, numAlreadyInUse int

	s.ilock.RLock()
	defer s.ilock.RUnlock()
	for name, intf := range s.interfaces {
		addrs, err := intf.Addrs()
		if err != nil {
			panic("could not get addresses from " + name + ": " + err.Error())
		}
		s.log.Debug("ip4v scanner scanning interface", "interface", name, "num_addrs", len(addrs))
		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPAddr:
				s.log.Warn("ipv4 scanner got a *net.IPAddr, which isn't useful and maybe shoudln't happen?", "interface", intf.Name, "*net.IPAddr", v)
			case *net.IPNet:
				if v.IP.DefaultMask() != nil { // ignore IPs without default mask (IPv6?)
					ip := v.IP

					for ip := ip.Mask(v.Mask); v.Contains(ip); incrementIP(ip) {
						wg.Add(1)
						numChecked++

						// To save space, try and only use 4 bytes
						if x := ip.To4(); x != nil {
							ip = x
						}
						dup := make(net.IP, len(ip)) // make a copy of the IP ([]byte)
						copy(dup, ip)
						go func() {
							defer wg.Done()
							s.slock.RLock()
							_, ok := s.activeServicesByIP[dup.String()] // ignore IPs already in use
							s.slock.RUnlock()

							if ok { // ignore IPs already in use
								s.log.Debug("scanner ignoring IP that is already in use", "ip", dup.String())
								numAlreadyInUse++
							} else {
								ids := s.getMatchingDescriptions(dup)
								if len(ids) > 0 { // At least one service matches
									s.log.Debug("found possible matches for ipv4 service", "num_matches", len(ids), "matching_ids", ids)
									numFound++
									flock.Lock()
									foundServices[dup.String()] = ids
									flock.Unlock()
									s.slock.Lock()
									s.activeServicesByIP[dup.String()] = struct{}{} // mark IP as in use
									s.slock.Unlock()
								}
							}
						}()
					}
				}
			default:
				s.log.Warn("ipv4 scanner encountered address of unknown type", "type", fmt.Sprintf("%T", a))
			}
		}
	}
	s.log.Debug("ipv4 scanner waiting for waitgroup to finish")
	wg.Wait()
	s.log.Debug("ipv4 scanner done waiting (all waitgroup items completed)")
	s.log.Info("ipv4 scan complete", "ips_checked", numChecked, "possibilities_found", numFound, "ips_already_in_use", numAlreadyInUse)
	return foundServices
}

// Unlock unlocks the provided IP, such that it will no longer be ignored in
// future scans.
func (s *Scanner) Unlock(ip net.IP) {
	s.slock.Lock()
	defer s.slock.Unlock()
	delete(s.activeServicesByIP, ip.String())
	s.log.Debug("ipv4 scanner unlocked IP", "ip", ip.String())
}

func (s *Scanner) getMatchingDescriptions(ip net.IP) []string {
	var matchedDrivers []string
	matchedPorts := make(map[uint16]bool) // true if found open, false if not, nil key if untested
	s.dlock.RLock()
	defer s.dlock.RUnlock()
	for id, desc := range s.descriptionsByID {
		match := true
		for _, port := range desc.OpenPorts {
			// Try the cache first
			if portIsOpen, ok := matchedPorts[port]; ok {
				if !portIsOpen {
					match = false
					s.log.Debug("ipv4 scanner found a service description which is not a match for services available at the target IP, because a port which is expected to be open is closed (according to cache)", "ip", ip.String(), "description_id", id, "port", port, "desc_ports", desc.OpenPorts)
					break
				}
			} else {
				// No cached entry, try dialing
				timeout := 1 * time.Second
				url := ip.String() + ":" + strconv.Itoa(int(port))
				conn, err := net.DialTimeout("tcp", url, timeout)
				if err != nil {
					match = false
					matchedPorts[port] = false
					s.log.Debug("ipv4 scanner found a service description which is not a match for the services available at the target IP, because a port which is expected to be open is closed (timed out trying)", "ip", ip.String(), "description_id", id, "port", port, "url", url, "timeout", timeout, "err", err, "desc_ports", desc.OpenPorts)
					break
				}
				conn.Close()
			}
		}
		if match {
			matchedDrivers = append(matchedDrivers, id) // add
			s.log.Debug("ipv4 scanner found a service description match", "ip", ip.String(), "service_desc", id)
		}
	}
	return matchedDrivers
}

// refreshInterfaces searches the device for any new or removed network interfaces
func (s *Scanner) refreshInterfaces() error {
	s.ilock.Lock() // Lock the interfaces for writing
	defer s.ilock.Unlock()

	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("error while refreshing ipv4 interfaces: %v", err)
	}
	foundInterfaces := make(map[string]net.Interface)
	newInterfaces := make(map[string]struct{})

	for _, iface := range ifaces {
		name := iface.Name
		if _, ok := ignoredIPv4Interfaces[name]; !ok { // ignore certain interfaces
			foundInterfaces[name] = iface
			if _, ok := s.interfaces[name]; !ok {
				newInterfaces[name] = struct{}{}
			}
			delete(s.interfaces, name) // unmark the interface - anything left once we're done has disappeared
		}
	}

	if len(s.interfaces) > 0 {
		names := make([]string, len(s.interfaces))
		i := 0
		for name := range s.interfaces {
			names[i] = name
		}
		s.log.Warn("STUB: IPv4 interfaces have disappeared, but handling logic is unimplemented! Services on the missing interfaces may still be active", "deleted_interfaces", names)
	}
	s.interfaces = foundInterfaces

	if len(newInterfaces) > 0 {
		names := make([]string, len(newInterfaces))
		i := 0
		for name := range newInterfaces {
			names[i] = name
			i++
		}
		s.log.Debug("new IPv4 interfaces found", "interfaces", names)
	}
	return nil
}

// incrementIP increments an IPv4 address
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// An IContinuousScanner is a Scanner that scans continuously. Scan results are
// passed to the channel provided by FoundServices()
type IContinuousScanner interface {
	suture.Service
	IScanner
	FoundServices() chan ServiceFoundNotification
}

// ContinuousScanner implements IContinuousScanner. It scans continuously,
// putting results into the channel provided by FoundServices().
type ContinuousScanner struct {
	*Scanner
	foundIPChan chan ServiceFoundNotification
	period      time.Duration
	stop        chan struct{}
}

// NewContinousScanner properly instantiates a ContinuousScanner. The new
// Scanner will wait between scans for a time defined by `period`.
func NewContinousScanner(period time.Duration) *ContinuousScanner {
	return &ContinuousScanner{
		Scanner:     NewScanner(),
		foundIPChan: make(chan ServiceFoundNotification),
		period:      period,
		stop:        make(chan struct{}),
	}
}

// FoundServices returns a channel which will be populated with services found
// by the ContinuousScanner
func (s *ContinuousScanner) FoundServices() chan ServiceFoundNotification {
	return s.foundIPChan
}

// Serve begins serving the ContinuousScanner.
func (s *ContinuousScanner) Serve() {
	s.log.Debug("starting continuous ipv4 scanner", "period", s.period)
	timer := time.NewTimer(time.Hour)
	for {
		// Perform a scan
		s.log.Debug("doing ipv4 scan")
		for ip, serviceIDs := range s.Scan() {
			s.log.Debug("found ipv4 scan", "ip", net.ParseIP(ip), "descriptions", serviceIDs)
			s.foundIPChan <- ServiceFoundNotification{
				IP: net.ParseIP(ip),
				MatchingDescriptionIDs: serviceIDs,
			}
		}

		// Wait for s.period
		s.log.Debug("waiting", "duration", s.period)
		timer.Reset(s.period)
		select {
		case <-s.stop:
			return
		case <-timer.C:
		}
	}
}

// Stop stops the ContinousScanner
func (s *ContinuousScanner) Stop() {
	s.stop <- struct{}{}
}
