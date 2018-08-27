package smartswitch

import (
	"errors"
	"fmt"

	"github.com/forfuncsake/minissdpc"
	"github.com/fromkeith/gossdp"
)

var (
	errNoSocket   = errors.New("no socket path provided for minissdp")
	errNoLocation = errors.New("no service location string provided")
)

type minissdp struct {
	socket string
	uuid   string
}

func (s minissdp) advertise(location string) error {
	if location == "" {
		return errNoLocation
	}
	if s.socket == "" {
		return errNoSocket
	}
	client := minissdpc.Client{SocketPath: s.socket}

	err := client.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to minissdp: %v", err)
	}

	return client.RegisterService(minissdpc.Service{
		Type:     "urn:Belkin:device:controllee:1",
		USN:      fmt.Sprintf("uuid:Socket-1_0-%s::urn:Belkin:device:controllee:1", s.uuid),
		Server:   "Forfuncsake SmartSwitch 1.0",
		Location: location,
	})
}

func (s minissdp) byebye() error {
	return nil
}

type rawssdp struct {
	uuid   string
	server *gossdp.Ssdp
}

func (s rawssdp) advertise(location string) error {
	if location == "" {
		return errNoLocation
	}

	var err error
	s.server, err = gossdp.NewSsdp(nil)
	if err != nil {
		return fmt.Errorf("could not create ssdp server: %v", err)
	}

	serverDef := gossdp.AdvertisableServer{
		ServiceType: "urn:Belkin:device:controllee:1",
		DeviceUuid:  s.uuid,
		Location:    location,
		MaxAge:      3600,
	}

	s.server.AdvertiseServer(serverDef)
	go s.server.Start()

	return nil
}

func (s rawssdp) byebye() error {
	if s.server != nil {
		s.server.Stop()
	}

	return nil
}
