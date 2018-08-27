package minissdpc

import (
	"bytes"
	"errors"
	"fmt"
	"net"
)

var (
	errNilConn = errors.New("client connection is nil")
	errOpen    = errors.New("attempted connect of an open client")
)

// DefaultSocket stores the regular path to the minissdpd unix socket
var DefaultSocket = "/var/run/minissdpd.sock"

// Client is used to interact with the minissdpd socket.
// Its methods are not safe for concurrent use, but can be
// re-used by calling Connect() after Close()
type Client struct {
	SocketPath string
	conn       net.Conn
}

// Close will close the underlying connection to the minissdpd socket
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	defer func() {
		c.conn = nil
	}()
	return c.conn.Close()
}

// Connect will open a connection to the minissdpd socket
func (c *Client) Connect() error {
	var err error
	if c.conn != nil {
		return errOpen
	}
	if c.SocketPath == "" {
		c.SocketPath = DefaultSocket
	}
	c.conn, err = net.Dial("unix", c.SocketPath)
	return err
}

// Write will attempt to write the provided byte slice
// onto the minissdpd socket
func (c *Client) Write(b []byte) (int, error) {
	if c.conn == nil {
		return 0, errNilConn
	}
	return c.conn.Write(b)
}

// WriteString will write a string onto the minissdpd socket
// prefixed with the encoded length bytes
func (c *Client) WriteString(s string) (int, error) {
	// Buffer the string with its encoded length prefixed
	buf := &bytes.Buffer{}
	err := EncodeStringLength(len(s), buf)
	if err != nil {
		return 0, fmt.Errorf("could not write string length byte(s): %v", err)
	}

	_, err = buf.WriteString(s)
	if err != nil {
		return 0, fmt.Errorf("could not write string to request buffer: %v", err)
	}

	// Send the request on the socket
	return c.Write(buf.Bytes())
}

// RegisterService will register a new service to be advertised
// by minissdpd
func (c *Client) RegisterService(s Service) error {
	b := bytes.NewBuffer([]byte{RequestTypeRegister})
	_, err := s.EncodeTo(b)
	if err != nil {
		return fmt.Errorf("could not encode service: %v", err)
	}

	_, err = c.Write(b.Bytes())
	return err
}

// GetServicesAll will query the minissdpd server for all services
// currently under advertisement
func (c *Client) GetServicesAll() ([]Service, error) {
	// Send the request on the socket
	_, err := c.Write([]byte{RequestTypeAll, 1, 0})
	if err != nil {
		return nil, fmt.Errorf("could not send request: %v", err)
	}

	// Decode the response
	return decodeServices(c.conn)
}

// GetServicesByUSN will query the minissdpd server for all services
// under advertisement that match the given USN string
func (c *Client) GetServicesByUSN(t string) ([]Service, error) {
	_, err := c.Write([]byte{RequestTypeByUSN})
	if err != nil {
		return nil, fmt.Errorf("could not start request: %v", err)
	}

	_, err = c.WriteString(t)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %v", err)
	}

	// Decode the response
	return decodeServices(c.conn)
}

// GetServicesByType will query the minissdpd server for all services
// under advertisement that match the given type string
func (c *Client) GetServicesByType(t string) ([]Service, error) {
	_, err := c.Write([]byte{RequestTypeByType})
	if err != nil {
		return nil, fmt.Errorf("could not start request: %v", err)
	}

	_, err = c.WriteString(t)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %v", err)
	}

	// Decode the response
	return decodeServices(c.conn)
}
