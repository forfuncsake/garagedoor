package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
)

// This is a value specified in the ESP8266 firmware
// to trigger an explicit button press, regardless
// of current door state.
const press int = -5

var stateURL = map[int]string{
	characteristic.TargetDoorStateOpen:   "/open",
	characteristic.TargetDoorStateClosed: "/close",
	press: "/press",
}

type apiResponse struct {
	Success bool   `json:"success"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// GarageDoor represents a HomeKit Accessory with a GarageDoorOpener
// and a Switch. The Opener will intelligently request a target state
// for the door (opened/closed), where the switch will always
// trigger the door button.
type GarageDoor struct {
	URL      string
	User     string
	Password string

	*accessory.Accessory
	Opener *service.GarageDoorOpener
	Button *service.Switch

	state      int
	guard      chan struct{}
	guardDelay time.Duration
}

// NewGarageDoor returns a GarageDoor with the provided accessory info.
func NewGarageDoor(conf Config, info accessory.Info) *GarageDoor {
	acc := GarageDoor{
		URL:       conf.URL,
		User:      conf.Username,
		Password:  conf.Password,
		Accessory: accessory.New(info, accessory.TypeGarageDoorOpener),
		Button:    service.NewSwitch(),
		Opener:    service.NewGarageDoorOpener(),
	}

	// Apply rate limiter, if configured
	if conf.Limit > 0 {
		acc.guardDelay = time.Duration(conf.Limit) * time.Second
		acc.guard = make(chan struct{}, 1)

		// Load a token into the guard channel
		acc.guard <- struct{}{}
	}

	acc.AddService(acc.Opener.Service)
	acc.AddService(acc.Button.Service)

	acc.Opener.CurrentDoorState.OnValueRemoteGet(acc.getState)
	acc.Opener.TargetDoorState.OnValueRemoteGet(acc.getTargetState)
	acc.Opener.TargetDoorState.OnValueRemoteUpdate(acc.setState)
	acc.Opener.CurrentDoorState.SetEventsEnabled(true)
	acc.Button.On.OnValueRemoteUpdate(acc.pressButton)

	return &acc
}

func (d *GarageDoor) pressButton(on bool) {
	if on {
		d.setState(press)

		// Always switch back to "off" to act like the
		// momentary switch this represents
		d.Button.On.SetValue(false)
	}
}

func (d *GarageDoor) setState(to int) {
	log.Printf("setState called")

	path, ok := stateURL[to]
	if !ok {
		log.Printf("unsupported state ID requested: %d", to)
		return
	}

	req, err := http.NewRequest(http.MethodPost, d.URL+path, nil)
	if err != nil {
		log.Printf("failed to create POST request: %v", err)
		return
	}

	req.SetBasicAuth(d.User, d.Password)
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("failed to post to button: %v", err)
	}
}

func (d *GarageDoor) getState() (state int) {
	defer func() {
		if state >= 0 {
			d.state = state
		} else {
			state = characteristic.CurrentDoorStateStopped
		}
	}()
	state = -1

	// Guard the module from being probed more than the configured limit
	// by returning the last saved state
	if d.guard != nil {
		select {
		case <-d.guard:
			go func() {
				<-time.After(d.guardDelay)
				d.guard <- struct{}{}
			}()
		default:
			log.Println("Guarding probe from excessive polls")
			return d.state
		}
	}

	resp, err := http.Get(d.URL)
	if err != nil {
		log.Printf("error getting status: %v\n", err)
		return state
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading status response: %v\n", err)
		return state
	}

	var msg apiResponse
	err = json.Unmarshal(b, &msg)
	if err != nil {
		log.Printf("error marshalling status response: %v\n", err)
		return state
	}

	if !msg.Success {
		log.Printf("got error from API: %s\n", msg.Message)
		if msg.Status < 1 {
			return state
		}
	}

	state = msg.Status
	return state
}

func (d *GarageDoor) getTargetState() (state int) {
	switch d.getState() {
	case characteristic.CurrentDoorStateClosed, characteristic.CurrentDoorStateClosing:
		return characteristic.TargetDoorStateClosed
	case characteristic.CurrentDoorStateOpen, characteristic.CurrentDoorStateOpening:
		return characteristic.TargetDoorStateOpen
	default:
		return characteristic.CurrentDoorStateStopped
	}
}
