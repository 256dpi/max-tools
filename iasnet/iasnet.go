package main

import (
	"errors"
	"net"
	"sync"

	"github.com/256dpi/max-go"
)

// TODO: Add heartbeat.

type object struct {
	stats   *max.Outlet
	local   *net.UDPConn
	router  *net.UDPConn
	targets []*net.UDPAddr
	closed  bool
	mutex   sync.Mutex
}

func (o *object) Init(obj *max.Object, args []max.Atom) bool {
	// add outlet
	o.stats = obj.Outlet(max.List, "bytes received/sent")

	// check args
	if len(args) < 5 || len(args)%2 != 1 {
		max.Error("iasnet: expected 5, 7, ... arguments")
		return false
	}

	// get local address
	local, err := net.ResolveUDPAddr("udp4", net.JoinHostPort("0.0.0.0", max.ToString(args[0])))
	if err != nil {
		max.Error("iasnet: %s", err.Error())
		return false
	}

	// get router address
	routerIP := max.ToString(args[1])
	router, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(routerIP, max.ToString(args[2])))
	if err != nil {
		max.Error("iasnet: %s", err.Error())
		return false
	}

	// get target addresses
	for i := 3; i < len(args); i += 2 {
		targetIP := max.ToString(args[i])
		targetAddr, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(targetIP, max.ToString(args[i+1])))
		if err != nil {
			max.Error("iasnet: %s", err.Error())
			return false
		}
		o.targets = append(o.targets, targetAddr)
	}

	// create local socket
	o.local, err = net.ListenUDP("udp4", local)
	if err != nil {
		max.Error("iasnet: %s", err.Error())
		return false
	}

	// create router socket
	o.router, err = net.DialUDP("udp4", nil, router)

	// handle sockets
	go o.relay()
	go o.distribute()

	return true
}

func (o *object) relay() {
	// prepare buffer
	buf := make([]byte, 2<<15) // 64k

	for {
		// read message
		n, err := o.local.Read(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			max.Error("iasnet: %s", err.Error())
		}

		// relay message
		_, err = o.router.Write(buf[:n])
		if err != nil {
			max.Error("iasnet: %s", err.Error())
		}
	}
}

func (o *object) distribute() {
	// prepare buffer
	buf := make([]byte, 2<<15) // 64k

	for {
		// read message
		n, err := o.router.Read(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			max.Error("iasnet: %s", err.Error())
		}

		// distribute message
		for _, target := range o.targets {
			_, err = o.local.WriteToUDP(buf[:n], target)
			if err != nil {
				max.Error("iasnet: %s", err.Error())
			}
		}
	}
}

func (o *object) Handle(_ int, _ string, data []max.Atom) {
	// ignore
}

func (o *object) Free() {
	// acquire mutex
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// close sockets
	_ = o.local.Close()
	_ = o.router.Close()

	// set flag
	o.closed = true
}

func main() {
	max.Register("iasnet", &object{})
}
