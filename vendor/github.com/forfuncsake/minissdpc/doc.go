// Package minissdpc provides an interface to interact with
// a running instance of minissdpd via its UNIX socket.
// It allows a go application to advertise it's service over
// SSDP when UDP Port 1900 is already in use by minissdpd, or
// to query minissdpd for all registered services.
//
// The implementation is based on the following information
// taken from: http://miniupnp.free.fr/minissdpd.html
//
//     Request are sent to the Unix socket. The first byte of the request is the request type.
//     Strings sent or recieved are not zero-terminated but prefixed by their length in a variable length format.
//     Use following macros to encode and decode to this format :
//
//         /* Encode length by using 7bit per Byte :
//          * Most significant bit of each byte specifies that the
//          * following byte is part of the code */
//         #define DECODELENGTH(n, p) n = 0; \
//             do { n = (n << 7) | (*p & 0x7f); } \
//             while(*(p++)&0x80);
//
//         #define CODELENGTH(n, p) if(n>=268435456) *(p++) = (n >> 28) | 0x80; \
//             if(n>=2097152) *(p++) = (n >> 21) | 0x80; \
//             if(n>=16384) *(p++) = (n >> 14) | 0x80; \
//             if(n>=128) *(p++) = (n >> 7) | 0x80; \
//             *(p++) = n & 0x7f;
package minissdpc
