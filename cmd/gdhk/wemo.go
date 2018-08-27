package main

import (
	"log"

	"github.com/brutella/hc/characteristic"
	"github.com/forfuncsake/smartswitch"
)

// true == Switch On == Open Door
// false == Switch off == Close Door
func boolToTargetState(v bool) int {
	if v {
		return characteristic.TargetDoorStateOpen
	}
	return characteristic.TargetDoorStateClosed
}

// 1 == Door Closed == Switch Off == false
// Otherwise the door is fully or partially open, so the switch shows "on"
func stateToBool(i int) bool {
	return i != characteristic.TargetDoorStateClosed
}

// Set changes the target state of the door
// satisfying the smartswitch.Switch interface
func (d *GarageDoor) Set(on bool) error {
	d.setState(boolToTargetState(on))
	return nil
}

// Status returns the status of the door (true=on=open, false=off=closed)
// satisfying the smartswitch.Switch interface
func (d *GarageDoor) Status() (bool, error) {
	return stateToBool(d.getState()), nil
}

func (d *GarageDoor) enableWemo() error {
	wemo := smartswitch.NewController(d.Name, d, smartswitch.WithMinissdpSocket("/var/run/minissdpd.sock"))
	loc, err := wemo.Start()
	if err != nil {
		return err
	}
	log.Printf("Wemo handler listening on %s\n", loc)
	return nil
}
