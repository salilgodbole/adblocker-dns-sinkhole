package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	logger := NewQueryLogger(1000)

	blocklist := NewBlocklist()
	if err := blocklist.Load(); err != nil {
		fmt.Println("Error loading blocklist:", err)
		os.Exit(1)
	}

	// Start watching the custom blocklist in the background
	go blocklist.WatchCustomBlocklist()

	// Start Admin Server
	adminServer := NewAdminServer(logger)
	go adminServer.Start("8333")

	addr := &net.UDPAddr{
		Port: 53,
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Failed to start DNS server (Do you need sudo?):", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("DNS Server listening on UDP port 53...")

	buf := make([]byte, 512)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			continue
		}

		// Handle request in a goroutine
		reqBuf := make([]byte, n)
		copy(reqBuf, buf[:n])
		go handleDNSRequest(conn, clientAddr, reqBuf, blocklist, logger)
	}
}
