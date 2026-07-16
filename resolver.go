package main

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const upstreamDNS = "1.1.1.1:53"

func handleDNSRequest(conn *net.UDPConn, addr *net.UDPAddr, buf []byte, blocklist *Blocklist) {
	var msg dnsmessage.Message
	if err := msg.Unpack(buf); err != nil {
		return
	}

	if len(msg.Questions) == 0 {
		return
	}

	question := msg.Questions[0]
	domainName := question.Name.String()

	if blocklist.IsBlocked(domainName) {
		fmt.Printf("Blocked: %s\n", domainName)
		sendBlockedResponse(conn, addr, msg)
		return
	}

	forwardToUpstream(conn, addr, buf)
}

func sendBlockedResponse(conn *net.UDPConn, addr *net.UDPAddr, req dnsmessage.Message) {
	req.Response = true
	req.RCode = dnsmessage.RCodeSuccess
	
	// Create a dummy A record pointing to 0.0.0.0
	answer := dnsmessage.Resource{
		Header: dnsmessage.ResourceHeader{
			Name:  req.Questions[0].Name,
			Type:  dnsmessage.TypeA,
			Class: dnsmessage.ClassINET,
			TTL:   60,
		},
		Body: &dnsmessage.AResource{
			A: [4]byte{0, 0, 0, 0},
		},
	}
	
	req.Answers = append(req.Answers, answer)

	packed, err := req.Pack()
	if err != nil {
		fmt.Println("Failed to pack response:", err)
		return
	}

	conn.WriteToUDP(packed, addr)
}

func forwardToUpstream(clientConn *net.UDPConn, clientAddr *net.UDPAddr, queryBuf []byte) {
	upstreamAddr, err := net.ResolveUDPAddr("udp", upstreamDNS)
	if err != nil {
		return
	}

	upstreamConn, err := net.DialUDP("udp", nil, upstreamAddr)
	if err != nil {
		return
	}
	defer upstreamConn.Close()
	
	upstreamConn.SetDeadline(time.Now().Add(2 * time.Second))

	if _, err := upstreamConn.Write(queryBuf); err != nil {
		return
	}

	respBuf := make([]byte, 512)
	n, err := upstreamConn.Read(respBuf)
	if err != nil {
		return
	}

	clientConn.WriteToUDP(respBuf[:n], clientAddr)
}
