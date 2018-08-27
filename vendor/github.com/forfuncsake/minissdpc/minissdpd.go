package minissdpc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// MaxLengthBytes determines the maximum number of bytes that can be used
// to encode the length. The default value of 5 theoretically allows a max
// length of 34359738367 (which overflows a 32bit int anyway)
const MaxLengthBytes = 5

// Request Types as defined by minissdpd
const (
	RequestTypeByType   byte = 1
	RequestTypeByUSN    byte = 2
	RequestTypeAll      byte = 3
	RequestTypeRegister byte = 4
)

var (
	errInvalidLength = errors.New("provided length is invalid")
	errNilWriter     = errors.New("received nil io.Writer")
	errTooLong       = errors.New("too many bytes read for string length")
)

func pow(x, y uint) uint {
	v := uint(1)
	for i := uint(0); i < y; i++ {
		v *= x
	}
	return v
}

// EncodeStringLength takes the length of a string as an integer
// and encodes it as a slice of bytes to the provided Writer.
func EncodeStringLength(length int, w io.Writer) error {
	if length < 0 {
		return errInvalidLength
	}
	if w == nil {
		return errNilWriter
	}

	n := uint(length)
	b := make([]byte, MaxLengthBytes)

	b[MaxLengthBytes-1] = byte(n & 0x7f)
	var i uint
	for i = 1; i < MaxLengthBytes; i++ {
		if n >= pow(128, i) {
			b[MaxLengthBytes-1-i] = byte(n>>(7*i) | 0x80)
		} else {
			break
		}
	}
	i--

	_, err := w.Write(b[MaxLengthBytes-1-i:])
	if err != nil {
		return fmt.Errorf("could not write to buffer: %v", err)
	}
	return nil
}

// DecodeStringLength reads the length bytes from the provided Reader
// and decodes them into the integer value.
func DecodeStringLength(r io.Reader) (int, error) {
	length := 0
	b := make([]byte, 1)

	for i := 1; ; i++ {
		if i > MaxLengthBytes {
			return 0, errTooLong
		}
		n, err := r.Read(b)
		if err != nil {
			return 0, fmt.Errorf("could not read buffer: %v", err)
		}
		if n != 1 {
			return 0, fmt.Errorf("expected to read 1 byte, got %d", n)
		}

		length = (length << 7) | int(b[0]&0x7f)

		if b[0]&0x80 != 0x80 {
			break
		}
	}

	return length, nil
}

// A Service represents an SSDP service that can be
// be advertised by minissdpd
type Service struct {
	Type     string
	USN      string
	Server   string
	Location string
}

// Encode will encode the service into a slice of bytes
// that can be written to the minissdpd socket
func (s *Service) Encode() ([]byte, error) {
	buf := &bytes.Buffer{}
	for _, v := range []string{
		s.Type,
		s.USN,
		s.Server,
		s.Location,
	} {
		err := EncodeStringLength(len(v), buf)
		if err != nil {
			return nil, fmt.Errorf("could not encode length of %q: %v", v, err)
		}
		_, err = buf.WriteString(v)
		if err != nil {
			return nil, fmt.Errorf("could not write string to buffer: %v", err)
		}
	}
	return buf.Bytes(), nil
}

// EncodeTo will encode and write the service as bytes, in
// the format required by minissdpd
func (s *Service) EncodeTo(w io.Writer) (int, error) {
	b, err := s.Encode()
	if err != nil {
		return 0, err
	}

	return w.Write(b)
}

func decodeServices(r io.Reader) ([]Service, error) {
	// The first byte is the number of services in the response
	buf := make([]byte, 1)
	_, err := r.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("could not read count from start of response: %v", err)
	}
	count := int(buf[0])

	services := make([]Service, count)
	for i := 0; i < count; i++ {
		var service Service
		for _, s := range []*string{&service.Location, &service.Type, &service.USN} {
			length, err := DecodeStringLength(r)
			if err != nil {
				return services, fmt.Errorf("error decoding string length: %v", err)
			}

			buf := make([]byte, length)
			n, err := r.Read(buf)
			if err != nil {
				return services, fmt.Errorf("error reading string: %v", err)
			}
			if n != length {
				return services, fmt.Errorf("expected to read %d bytes, got %d", length, n)
			}
			*s = string(buf)
		}
		services[i] = service
	}

	return services, nil
}
