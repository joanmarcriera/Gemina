// Package exit implements the gateway's exit path: it takes decrypted inner IP
// packets from admitted sessions and forwards them to a TUN device, then reads
// return traffic from that device and routes it back to every known source
// endpoint for the originating session.
//
// The package is designed so all I/O is injected through small interfaces
// (Device, Framer, Sink), making the routing logic fully testable with fakes.
package exit
