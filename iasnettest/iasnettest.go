package main

import (
	"flag"
	"net"
	"net/url"
)

var addr = flag.String("addr", "0.0.0.0:2345", "")

func main() {
	flag.Parse()

	udpAddr, err := net.ResolveUDPAddr("udp", *addr)
	if err != nil {
		panic(err)
	}

	sock, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	buf := make([]byte, 1<<16) // 64k

	for {
		n, addr, err := sock.ReadFromUDP(buf)
		if err != nil {
			panic(err)
		}

		_, err = sock.WriteToUDP(buf[:n], addr)
		if err != nil {
			panic(err)
		}

		println("=> " + url.QueryEscape(string(buf[:n])))
	}
}
