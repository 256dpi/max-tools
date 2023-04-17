package main

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/256dpi/max-go"
)

type object struct {
	stats   *max.Outlet
	dump    *max.Outlet
	targets []*net.UDPAddr
	local   *net.UDPConn
	router  *net.UDPConn
	done    chan struct{}
	in      int64
	out     int64
	errMap  map[string]int64
	mutex   sync.Mutex
}

func (o *object) Init(obj *max.Object, args []max.Atom) bool {
	// add outlets
	o.stats = obj.Outlet(max.List, "bytes received/sent")
	o.dump = obj.Outlet(max.List, "dump messages")

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
	err = adjustBuffers(o.local, 1<<20)
	if err != nil {
		max.Error("iasnet: %s", err.Error())
		return false
	}

	// create router socket
	o.router, err = net.DialUDP("udp4", nil, router)
	if err != nil {
		max.Error("iasnet: %s", err.Error())
		return false
	}
	err = adjustBuffers(o.router, 1<<20)
	if err != nil {
		max.Error("iasnet: %s", err.Error())
		return false
	}

	// create signal
	o.done = make(chan struct{})

	// handle sockets
	go o.relay()
	go o.distribute()
	go o.manage()

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
			o.incError(err)
		}

		// relay message
		_, err = o.router.Write(buf[:n])
		if err != nil {
			o.incError(err)
		}

		// increment counter
		atomic.AddInt64(&o.in, int64(n))
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
			o.incError(err)
		}

		// distribute message
		for _, target := range o.targets {
			_, err = o.local.WriteToUDP(buf[:n], target)
			if err != nil {
				o.incError(err)
			}
		}

		// increment counter
		atomic.AddInt64(&o.out, int64(n))
	}
}

func (o *object) manage() {
	// prepare heartbeat
	hb := []byte{47, 104, 98, 0, 44, 0, 0, 0}

	for {
		// wait a bit
		select {
		case <-o.done:
			return
		case <-time.After(time.Second):
		}

		// send heartbeat
		_, err := o.router.Write(hb)
		if err != nil {
			o.incError(err)
		}

		// gather statistics
		in := atomic.SwapInt64(&o.in, 0)
		out := atomic.SwapInt64(&o.out, 0)

		// send statistics
		o.stats.List([]max.Atom{in, out})

		// get errors
		o.mutex.Lock()
		errMap := o.errMap
		o.errMap = map[string]int64{}
		o.mutex.Unlock()

		// send errors
		for str, num := range errMap {
			o.dump.List([]max.Atom{"error", str, num})
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

	// close signal
	close(o.done)
}

func (o *object) incError(err error) {
	// increment error
	o.mutex.Lock()
	o.errMap[err.Error()]++
	o.mutex.Unlock()
}

func main() {
	max.Register("iasnet", &object{})
}

func adjustBuffers(conn *net.UDPConn, size int) error {
	// set write buffer
	err := conn.SetWriteBuffer(size)
	if err != nil {
		return err
	}

	// set read buffer
	err = conn.SetReadBuffer(size)
	if err != nil {
		return err
	}

	return nil
}
