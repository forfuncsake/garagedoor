package smartswitch

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
)

const (
	serialChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	serialLength = 8
)

const (
	setupURI     = "/setup.xml"
	controlURI   = "/upnp/control/basicevent1"
	subscribeURI = "/upnp/event/basicevent1"
)

const (
	wemoSetupFormat = `<?xml version="1.0"?>
	<root xmlns="urn:Belkin:device-1-0">
		<specVersion>
		<major>1</major>
		<minor>0</minor>
		</specVersion>
		<device>
			<deviceType>urn:Belkin:device:controllee:1</deviceType>
			<friendlyName>%s</friendlyName>
			<manufacturer>Belkin International Inc.</manufacturer>
			<modelName>Emulated Socket</modelName>
			<modelNumber>3.1415</modelNumber>
			<manufacturerURL>http://www.belkin.com</manufacturerURL>
			<modelDescription>Belkin Plugin Socket 1.0</modelDescription>
			<modelURL>http://www.belkin.com/plugin/</modelURL>
			<UDN>uuid:%s</UDN>
			<serialNumber>%s</serialNumber>
			<binaryState>0</binaryState>
			<serviceList>
				<service>
					<serviceType>urn:Belkin:service:basicevent:1</serviceType>
					<serviceId>urn:Belkin:serviceId:basicevent1</serviceId>
					<controlURL>%s</controlURL>
					<eventSubURL>%s</eventSubURL>
				</service>
			</serviceList>
		</device>
	</root>`

	wemoResponseFormat = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
		<s:Body>
			<u:%[1]sBinaryStateResponse xmlns:u="urn:Belkin:service:basicevent:1">
				<BinaryState>%d</BinaryState>
			</u:%[1]sBinaryStateResponse>
		</s:Body>
	</s:Envelope>`
)

var (
	errNoServer = errors.New("controller server not initialised, must call NewController(...) before Start()")
	errNoSwitch = errors.New("controller switch not initialised, must call Newcontroller(...) before Start()")
)

type advertiser interface {
	advertise(location string) error
	byebye() error
}

type server interface {
	Serve(net.Listener) error
	Close() error
}

// A Controller manages SSDP advertisement and client interactions
// for a Switch
type Controller struct {
	ssdp     advertiser
	srv      server
	listener net.Listener

	sw        Switch
	uriPrefix string
	socket    string
	iface     string
	addr      net.IP
	port      uint
	name      string
	uuid      string
	serial    string
}

// The Switch interface must be implemented by the device
// that you wish to control with Wemo clients
type Switch interface {
	Status() (bool, error)
	Set(bool) error
}

// ControllerOption values are accepted by NewController for custom configurations
type ControllerOption func(*Controller)

// WithListenAddress allows the caller to specify a single listen address
// for the controller to use (default is all IP addresses)
func WithListenAddress(addr net.IP) ControllerOption {
	return func(c *Controller) {
		c.addr = addr
	}
}

// WithListenPort allows the caller to specify a strict TCP port to use for
// services provided by the Controller (default is system-assigned ephemeral port)
func WithListenPort(p uint) ControllerOption {
	return func(c *Controller) {
		c.port = p
	}
}

// WithInterface allows the caller to specify a network interface to use for the Controller.
// The Controller will advertise services on the first IPv4 address of this interface
func WithInterface(name string) ControllerOption {
	return func(c *Controller) {
		c.iface = name
	}
}

// WithUUID allows the caller to specify a UUID value for the emulated switch (default: generated ID)
func WithUUID(s string) ControllerOption {
	return func(c *Controller) {
		c.uuid = s
	}
}

// WithMinissdpSocket is used to configure the controller to advertise via the Minissdpd service
// instead of starting an SSDP server (for Synology device support)
func WithMinissdpSocket(path string) ControllerOption {
	return func(c *Controller) {
		c.socket = path
	}
}

// WithURIPrefix allows the caller to configure a path prefix for the Wemo control URIs
func WithURIPrefix(s string) ControllerOption {
	return func(c *Controller) {
		c.uriPrefix = s
	}
}

// NewController creates a smart switch controller with the specified options
func NewController(name string, s Switch, opts ...ControllerOption) *Controller {
	serial := make([]byte, serialLength)
	for i := 0; i < serialLength; i++ {
		serial[i] = serialChars[rand.Int()%len(serialChars)]
	}

	if name == "" {
		name = "smartswitch-" + string(serial)
	}

	c := &Controller{
		name:   name,
		serial: string(serial),
		sw:     s,
	}

	for _, o := range opts {
		o(c)
	}

	if c.uuid == "" {
		c.uuid = uuid.NewV3(uuid.NamespaceURL, "smartswitch"+name).String()
	}

	if c.socket != "" {
		c.ssdp = minissdp{
			socket: c.socket,
			uuid:   c.uuid,
		}
	} else {
		c.ssdp = rawssdp{
			uuid: c.uuid,
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc(c.uriPrefix+setupURI, c.handleWemoSetup)
	mux.HandleFunc(c.uriPrefix+controlURI, c.handleWemoControl)

	timeout := 5 * time.Second
	c.srv = &http.Server{
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
		IdleTimeout:  timeout,
		Handler:      mux,
	}

	return c
}

// Start will create and serve an HTTP service to manage a smart switch
// and advertise the service using SSDP
func (c *Controller) Start() (location string, err error) {
	if c.srv == nil {
		return "", errNoServer
	}
	if c.sw == nil {
		return "", errNoSwitch
	}

	listenAddr := c.addr.To4()
	if listenAddr.IsUnspecified() {
		listenAddr = nil
	}

	// Get advertisable address (optionally filtered by interface)
	advAddr, err := advertiseIPAddr(c.iface)
	if err != nil {
		return "", err
	}

	ip := ""
	if listenAddr != nil {
		ip = listenAddr.String()
	} else if c.iface != "" {
		ip = advAddr.String()
	}

	// Create listener explicitly, so we can find the allocated port
	l, err := net.Listen("tcp", net.JoinHostPort(ip, strconv.Itoa(int(c.port))))
	if err != nil {
		return "", fmt.Errorf("could not start tcp listener: %v", err)
	}

	// Get final listener address (as it may not have been requested)
	parts := strings.Split(l.Addr().String(), ":")
	port := parts[len(parts)-1]

	location = fmt.Sprintf("http://%s:%s%s%s", advAddr.String(), port, c.uriPrefix, setupURI)

	go func() {
		err := c.srv.Serve(l)
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// return location for optional logging, advertise location with ssdp
	return location, c.ssdp.advertise(location)
}

// Stop will close the HTTP server and stop advertising over SSDP
func (c *Controller) Stop() error {
	if c.ssdp != nil {
		err := c.ssdp.byebye()
		if err != nil {
			return fmt.Errorf("could not send SSDP byebye message: %v", err)
		}
		c.ssdp = nil
	}

	if c.srv != nil {
		err := c.srv.Close()
		if err != nil {
			return fmt.Errorf("could not stop http server: %v", err)
		}
		c.srv = nil
	}

	if c.listener != nil {
		err := c.listener.Close()
		if err != nil {
			return fmt.Errorf("could not stop listener: %v", err)
		}
		c.listener = nil
	}

	return nil
}

func (c *Controller) handleWemoSetup(w http.ResponseWriter, r *http.Request) {
	//c.debug(r)
	w.Header().Set("Content-Type", "text/xml")
	fmt.Fprintf(w, wemoSetupFormat, c.name, c.uuid, c.serial, controlURI, subscribeURI)
}

func (c *Controller) handleWemoControl(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	r.Body.Close()
	body := string(b)

	responseType := "Get"
	if strings.Contains(body, "SetBinaryState") {
		state := false
		if strings.Contains(body, "<BinaryState>1</BinaryState>") {
			state = true
		}
		c.sw.Set(state)
		responseType = "Set"
	}

	state, err := c.sw.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get switch status: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/xml")
	fmt.Fprintf(w, wemoResponseFormat, responseType, toBinary(state))
}

func toBinary(v bool) int {
	if v {
		return 1
	}
	return 0
}

func advertiseIPAddr(iface string) (net.IP, error) {
	var addrs []net.Addr
	var err error

	if iface != "" {
		var i *net.Interface
		i, err = net.InterfaceByName(iface)
		if err != nil {
			return nil, fmt.Errorf("could not get addresses for %s: %v", iface, err)
		}
		addrs, err = i.Addrs()
	} else {
		addrs, err = net.InterfaceAddrs()
	}
	if err != nil {
		return nil, fmt.Errorf("could not get local network addresses: %v", err)
	}

	var ip net.IP
	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP.To4()
		case *net.IPAddr:
			ip = v.IP.To4()
		}
		if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
			continue
		}
		return ip, nil
	}

	return nil, errors.New("could not discover IPv4 address to advertise")
}

func (c *Controller) debug(r *http.Request) {
	fmt.Printf("%s -> %s\n", r.RemoteAddr, r.RequestURI)
}
