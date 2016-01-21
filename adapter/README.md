# sift drivers

SIFT adapters facilitate two-way sync between the SIFT server and networked
devices.

## Lifecycle
### Birth
* An AdapterFactory is added to a SIFT Server. The AdapterFactory describes the
  types of services that it is looking for. The SIFT Server begins looking for
  services matching that description.
* When a matching service is found, the SIFT Server spawns a new Adapter via the
  AdapterFactory.
* While the Adapter is working as intended, it sends periodic heartbeat signals
  back to the SIFT Server.

### device --> SIFT
* The new Adapter begins communicating with the service to determine the state
  of the Device (or Devices) which it represents. It may poll regularly for
  new data, or subscribe for updates if the service supports it. When the data
  is received, it is packed into an update and passed back to the SIFT Server
  for processing. So long as the Adapter is working as intended, it must send
  periodic heartbeat signals back to the SIFT Server.
* The SIFT Server consumes updates from the Adapter and reflects them in it's
  internal database.

### SIFT --> device
* SIFT Apps submit intents to the SIFT Server. SIFT determines the appropriate
  Adapter to handle them and passes them along.
* The Adapter communicates with the service to enact the intent.

### Death
* The SIFT Server waits until the Adapter is dead (either it reports an error or
  it misses a heartbeat interval). The SIFT Server disables the dead Adapter and
  frees up the network service to be found again on the next network scan.

  _Note: a released network service is often picked up on the next run by the
    same AdapterFactory, and a new version of the same Adapter is spawned. This
    is fine. From the perspective of the SIFT Server, this behavior
    both recovers from temporary network failures and adapts to changes in
    network services._

## Interface
Adapters must meet the Adapter interface

AdapterFactories must meet the AdapterFactory interface

AdapterFactories are distinguished by the networks on which they operate. An
AdapterFactory that produces Adapters that operate on IPv4 services must also
implement IPv4AdapterFactory.

See https://github.com/upwrd/sift/tree/master/adapter/example for a complete
example.
