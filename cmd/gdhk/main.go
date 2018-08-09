package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/kelseyhightower/envconfig"
)

var version = "develop"

// Config is used as a value store for envconfig
// (configuration via environment variables)
type Config struct {
	URL       string
	ProxyPort uint `default:"8180"`
	AccPort   uint

	Name        string `default:"GarageDoor"`
	Serial      string `default:"GDOOR-0001"`
	PIN         string `default:"12344321"`
	StoragePath string
	Username    string `default:"admin"`
	Password    string `default:"password"`
	Limit       uint
}

func main() {
	conf := Config{}
	err := envconfig.Process("gd", &conf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read config from environment: %v\n", err)
		os.Exit(1)
	}

	flag.StringVar(&conf.URL, "url", conf.URL, "URL for the garage door API")
	flag.UintVar(&conf.ProxyPort, "proxy-port", conf.ProxyPort, "TCP port for callback listener of this proxy")
	flag.UintVar(&conf.AccPort, "acc-port", conf.AccPort, "TCP port to use for HomeKit accessory")
	flag.StringVar(&conf.Name, "name", conf.Name, "Name of the HomeKit accessory")
	flag.StringVar(&conf.Serial, "serial", conf.Serial, "Serial number override")
	flag.StringVar(&conf.PIN, "pin", conf.PIN, "HomeKit setup code/PIN for this accessory")
	flag.StringVar(&conf.StoragePath, "path", conf.StoragePath, "Storage path for HomeKit pairing database")
	flag.StringVar(&conf.Username, "u", conf.Username, "`username` for requests to garage door API")
	flag.StringVar(&conf.Password, "p", conf.Password, "`password` for requests to garage door API")
	flag.UintVar(&conf.Limit, "limit", conf.Limit, "Limit probing the API to once every `n` seconds")
	e := flag.Bool("e", false, "show envconfig help and exit")
	v := flag.Bool("version", false, "show version and exit")
	flag.Parse()

	if *v {
		fmt.Printf("%s: %s", os.Args[0], version)
		os.Exit(0)
	}

	if *e {
		envconfig.Usage("gd", &Config{})
		os.Exit(0)
	}

	if conf.URL == "" {
		fmt.Fprintln(os.Stderr, "URL for garage door must be specified")
		flag.Usage()
		os.Exit(1)
	}

	u, err := url.Parse(conf.URL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "URL value is invalid: %q\n", conf.URL)
		os.Exit(1)
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	conf.URL = u.String()

	if conf.ProxyPort == 0 {
		fmt.Fprintln(os.Stderr, "Proxy port must be specified (non-zero)")
		flag.Usage()
		os.Exit(1)
	}

	info := accessory.Info{
		Name:         conf.Name,
		SerialNumber: conf.Serial,
		Manufacturer: "forfuncsake",
		Model:        "GDHK",
	}
	door := NewGarageDoor(conf, info)

	config := hc.Config{
		Pin:         conf.PIN,
		StoragePath: conf.StoragePath,
		Port:        strconv.Itoa(int(conf.AccPort)),
	}
	t, err := hc.NewIPTransport(config, door.Accessory)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create IP transport: %v\n", err)
		os.Exit(1)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/refresh", func(http.ResponseWriter, *http.Request) {
		door.getState()
		door.Opener.CurrentDoorState.SetValue(door.state)
		door.Opener.TargetDoorState.SetValue(door.getTargetState())
	})

	timeout := 5 * time.Second
	srv := http.Server{
		Addr:         fmt.Sprintf(":%d", conf.ProxyPort),
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
		IdleTimeout:  timeout,
		Handler:      mux,
	}

	go func() {
		srv.ListenAndServe()
	}()

	t.Start()
}
