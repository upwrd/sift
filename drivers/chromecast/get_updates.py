'''
get_updates will listen to a Chromecast adapter and print results to stdout
'''

import json

import sys
import time
import pychromecast

"usage: get_updates.py [IP address of Chromecast] [Time between polls]"

DEFAULT_TIME_BETWEEN_POLLS = 5 # seconds
EXIT_NOT_ENOUGH_PARAMS = -1
EXIT_CHROMECAST_NOT_FOUND = -2
EXIT_INVALID_ARGUMENTS = -3

zmqHeartbeatSocketAddr = "tcp://localhost:5557"
zmqUpdateSocketAddr    = "tcp://localhost:5558"

# These types are used to signal to a listener the format of a marshalled message
TYPE_ERROR = "error"
TYPE_UPDATE = "update"

# Use this when signaling an error
class Error():
    def __init__(self, error):
        self.type = TYPE_ERROR
        self.error = error
    def reprJSON(self):
        return self.__dict__

# A Update is sent each time the Chromecast's state changes
# It should match a MediaPlayer struct defined in sift/types
class Update():
    def __init__(self, update):
        self.type = TYPE_UPDATE
        self.update = update
    def reprJSON(self):
        return dict(type=self.type, update=self.update)

class MediaPlayerComponent():
    def __init__(self, ip, play_state, source):
        self.external_id=ip # Use the IP address as the external id
        self.make="Google"
        self.model="Chromecast"
        self.state = MediaPlayerState(play_state, source)
    def reprJSON(self):
        return self.__dict__

class MediaPlayerState():
    def __init__(self, play_state, source):
        self.play_state=play_state
        self.media_type="VIDEO"
        self.source=source
    def reprJSON(self):
        return self.__dict__

class ComplexEncoder(json.JSONEncoder):
    def default(self, obj):
        if hasattr(obj,'reprJSON'):
            return obj.reprJSON()
        else:
            return json.JSONEncoder.default(self, obj)

# Return an error if the caller does not supply an IP address
if len(sys.argv) < 2:
    err = Error("USAGE: sift_adapter.py [IP address] ([seconds between polls])")
    print(json.dumps(err.__dict__))
    sys.stdout.flush()
    exit(EXIT_NOT_ENOUGH_PARAMS)

ip = sys.argv[1] # use the IP that was passed in
if len(sys.argv) > 2:
    try:
        seconds_between_polls = float(sys.argv[2]) # if they passed in a 3rd argument, use it as a time between polls
    except:
        err = Error("third argument (seconds between polls) must be numerical")
        print(json.dumps(err.__dict__))
        sys.stdout.flush()
        exit(EXIT_INVALID_ARGUMENTS)
else:
    seconds_between_polls = DEFAULT_TIME_BETWEEN_POLLS # if not, use the default

cast = pychromecast.get_chromecast(ip=ip)
if cast is None:
    err = Error("could not find a Chromecast matching ip " + ip)
    print(json.dumps(err.__dict__))
    sys.stdout.flush()
    exit(EXIT_CHROMECAST_NOT_FOUND)
cast.wait() # be sure to wait until it is ready

while True:
    try:
        # Build a new update from the current state reported by the
        play_state = cast.media_controller.status.player_state
        source = cast.status.display_name
        new_update = Update(MediaPlayerComponent(ip, play_state, source))
        print(json.dumps(new_update.__dict__, cls=ComplexEncoder))
        sys.stdout.flush()
        time.sleep(seconds_between_polls) # sleep before polling again
    except Exception as e:
        err = Error("error trying to read Chromecast state " + e.message)
        print(json.dumps(err.__dict__))
        sys.stdout.flush()
        exit(EXIT_CHROMECAST_NOT_FOUND)
