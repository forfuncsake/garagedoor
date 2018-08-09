package main

import (
	"fmt"
	"testing"

	"github.com/kelseyhightower/envconfig"

	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
)

func newDoor(port int) *GarageDoor {
	conf := Config{}
	envconfig.Process("gd_test", &conf)
	conf.URL = fmt.Sprintf("http://127.0.0.1:%d", port)
	door := NewGarageDoor(conf, accessory.Info{
		Name:         "GarageDoorTest",
		SerialNumber: "1234567890",
		Manufacturer: "forfuncsake",
		Model:        "GDHK",
	})

	return door
}

func TestGetState(t *testing.T) {
	a, err := newAPI()
	if err != nil {
		t.Fatalf("could not start mock API: %v", err)
	}
	defer a.Close()
	door := newDoor(a.port)

	state := door.getState()
	if state != characteristic.TargetDoorStateOpen {
		t.Errorf("unexpected initialization state. expected %d, got: %d", characteristic.TargetDoorStateClosed, state)
	}
}

func TestPressButton(t *testing.T) {
	a, err := newAPI()
	if err != nil {
		t.Fatalf("could not start mock API: %v", err)
	}
	defer a.Close()
	door := newDoor(a.port)

	// "switch" should be off excepted while actively being "pressed"
	if door.Button.On.GetValue() {
		t.Fatalf("switch is unexpectedly on")
	}

	// Attempting to turn the switch of should have no impact
	door.pressButton(false)

	// validate that the mock API has zero presses
	if a.pressed > 0 {
		t.Fatalf("mock API has registered %d presses, expected none", a.pressed)
	}

	for i := 1; i < 5; i++ {
		door.pressButton(true)
		if a.pressed != i {
			t.Fatalf("expected %d presses on the API, detected: %d", i, a.pressed)
		}

		// "switch" should still be off
		if door.Button.On.GetValue() {
			t.Fatalf("switch is unexpectedly on")
		}
	}
}

func TestSetState(t *testing.T) {
	a, err := newAPI()
	if err != nil {
		t.Fatalf("could not start mock API: %v", err)
	}
	defer a.Close()
	door := newDoor(a.port)

	tests := []struct {
		Name  string
		State int
	}{
		{"Open", 0},
		{"Close", 1},
		{"Re-open", 0},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			door.setState(test.State)
			state := door.getState()
			if state != test.State {
				t.Errorf("unexpected door state. expected %d, got: %d", test.State, state)
			}
		})
	}
}
