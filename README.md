# Garage Door

Garage Door is a project for creating a DIY Smart Garage Door controller, with HomeKit compatibility.

## Getting Started

This project includes all of the software requirements for creating a DIY Smart Garage Door Controller based on the NodeMCU ESP8266 Development Board.  
For details on the hardware build, see my [write up](https://forfuncsake.github.io/2018/03/diy-smart-garage-door--part-1/).
  
This initial version of the project consists of 3 parts:
- GarageDoor.ino: The Arduino IDE project containing the firmware for the ESP8266 board
- cmd/gdhk: Go application providing a HomeKit proxy
- spksrc/spk/garagedoor: Quick packaging solution for installation of the proxy on a Synology NAS
  
It is currently very specific to the author's use case, without many configuration options. Check the [issues list](https://github.com/forfuncsake/garagedoor/issues) to see known issues and roadmap items.  

### Prerequisites

- [Arduino IDE](https://www.arduino.cc/en/Main/Software)
- [ESP8266 Packages](https://randomnerdtutorials.com/how-to-install-esp8266-board-arduino-ide/)
- [go](https://golang.org/dl/)

Optional:
- [spksrc](https://github.com/SynoCommunity/spksrc)

### Building/Installing

TODO

## Deployment

TODO

## Built With

* [hc](https://github.com/brutella/hc) - HomeControl is an implementation of the HomeKit Accessory Protocol (HAP) in Go
* [Arduino IDE](https://www.arduino.cc/en/Main/Software) - Build and flash ESP8266 Dev Board firmware
* [Arduino ESP8266 Add-On](http://esp8266.github.io/Arduino/versions/2.0.0/doc/installing.html) - Libraries for using ESP8266 boards in Arduino IDE
* [spksrc](https://github.com/SynoCommunity/spksrc) - Packaging the go binary for installation on Synology NAS

## Contributing

TODO

## Author

Dave Russell - [*forfuncsake*](https://github.com/forfuncsake/)
* https://forfuncsake.github.io/
* [@forfuncsake](https://twitter.com/forfuncsake)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details


[![Build Status](https://travis-ci.org/forfuncsake/garagedoor.svg?branch=master)](https://travis-ci.org/forfuncsake/garagedoor)