package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"syscall"
	"time"
)

func getSocketAddr() (addr [4]byte, err error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if len(ipnet.IP.To4()) == net.IPv4len {
				copy(addr[:], ipnet.IP.To4())
				return
			}
		}
	}
	err = errors.New("Cannot get local interface")
	return
}

// TracerouteOptions describes options for Traceroute
type TracerouteOptions struct {
	Port       int
	MaxTTL     int
	TimeoutMs  int
	Retries    int
	PacketSize int
}

// NewTracerouteOptions returns new TracerouteOptions with sane defaults
func NewTracerouteOptions() *TracerouteOptions {
	return &TracerouteOptions{
		Port:       33434,
		MaxTTL:     64,
		TimeoutMs:  500,
		Retries:    3,
		PacketSize: 52,
	}
}

// TraceUpdate contains data aout the udpate received from remote host
type TraceUpdate struct {
	Success     bool
	Address     [4]byte
	Host        string
	N           int
	ElapsedTime time.Duration
	TTL         int
}

func (hop *TraceUpdate) addressString() string {
	return fmt.Sprintf("%v.%v.%v.%v", hop.Address[0], hop.Address[1], hop.Address[2], hop.Address[3])
}

// Traceroute executes traceroute to given destination, using options from TracerouteOptions
// and sending updates to chan c
//
// Outbound packets are UDP packets and inbound packets are ICMP.
//
// Returns an error or nil if no error occurred
func Traceroute(dest *net.IPAddr, options *TracerouteOptions, c chan TraceUpdate) (err error) {
	var destAddr [4]byte
	copy(destAddr[:], dest.IP.To4())
	socketAddr, err := getSocketAddr()
	if err != nil {
		return
	}

	timeoutMs := (int64)(options.TimeoutMs)
	tv := syscall.NsecToTimeval(1000 * 1000 * timeoutMs)

	ttl := 1
	for {
		// Set up receiving socket
		recvSocket, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
		if err != nil {
			log.Fatal("Cannot setup receive socket, please run as root or with CAP_NET_RAW permissions")
			return err
		}
		// Set up sending socket
		sendSocket, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
		if err != nil {
			log.Fatal("Cannot setup sending socket")
			return err
		}

		start := time.Now()
		syscall.SetsockoptInt(sendSocket, 0x0, syscall.IP_TTL, ttl)
		syscall.SetsockoptTimeval(recvSocket, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv)
		syscall.Bind(recvSocket, &syscall.SockaddrInet4{Port: options.Port, Addr: socketAddr})
		syscall.Sendto(sendSocket, []byte{0x0}, 0, &syscall.SockaddrInet4{Port: options.Port, Addr: destAddr})

		var p = make([]byte, options.PacketSize)
		n, from, err := syscall.Recvfrom(recvSocket, p, 0)
		elapsed := time.Since(start)
		if err == nil {
			currAddr := from.(*syscall.SockaddrInet4).Addr
			hop := TraceUpdate{Success: true, Address: currAddr, N: n, ElapsedTime: elapsed, TTL: ttl}
			currHost, err := net.LookupAddr(hop.addressString())
			if err == nil {
				hop.Host = currHost[0]
			}
			// Send update
			c <- hop
			ttl += 1
			// We reached the destination
			if ttl > options.MaxTTL || currAddr == destAddr {
				ttl = 1
			}
		} else {
			c <- TraceUpdate{Success: false, TTL: ttl}
			ttl += 1
		}
		syscall.Close(recvSocket)
		syscall.Close(sendSocket)
	}
}
