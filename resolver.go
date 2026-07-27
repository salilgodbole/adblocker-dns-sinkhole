package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const upstreamDNS = "1.1.1.1:53"

func handleDNSRequest(conn *net.UDPConn, addr *net.UDPAddr, buf []byte, blocklist *Blocklist, logger *QueryLogger) {
	var msg dnsmessage.Message
	if err := msg.Unpack(buf); err != nil {
		return
	}

	if len(msg.Questions) == 0 {
		return
	}

	question := msg.Questions[0]
	domainName := question.Name.String()

	var queryType, responseStr string
	if logger.IsTrackingEnabled() {
		queryType = question.Type.String()
	}

	isBlocked, status := blocklist.IsBlocked(domainName)

	if isBlocked {
		if logger.IsTrackingEnabled() {
			responseStr = "0.0.0.0"
		}
		logger.Log(addr.IP.String(), domainName, status, queryType, responseStr)
		fmt.Printf("[%s] Blocked: %s\n", status, domainName)
		sendBlockedResponse(conn, addr, msg)
		return
	}

	respStr := forwardToUpstream(conn, addr, buf)
	if logger.IsTrackingEnabled() {
		responseStr = respStr
	}
	logger.Log(addr.IP.String(), domainName, status, queryType, responseStr)
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

func forwardToUpstream(clientConn *net.UDPConn, clientAddr *net.UDPAddr, queryBuf []byte) string {
	upstreamAddr, err := net.ResolveUDPAddr("udp", upstreamDNS)
	if err != nil {
		return ""
	}

	upstreamConn, err := net.DialUDP("udp", nil, upstreamAddr)
	if err != nil {
		return ""
	}
	defer upstreamConn.Close()
	
	upstreamConn.SetDeadline(time.Now().Add(2 * time.Second))

	if _, err := upstreamConn.Write(queryBuf); err != nil {
		return ""
	}

	respBuf := make([]byte, 512)
	n, err := upstreamConn.Read(respBuf)
	if err != nil {
		return ""
	}

	clientConn.WriteToUDP(respBuf[:n], clientAddr)

	var respStr string
	var respMsg dnsmessage.Message
	if err := respMsg.Unpack(respBuf[:n]); err == nil {
		var answers []string
		for _, ans := range respMsg.Answers {
			// A bit of cleanup for the raw output
			ansStr := fmt.Sprintf("%v", ans.Body)
			ansStr = strings.TrimPrefix(ansStr, "&")
			answers = append(answers, ansStr)
		}
		respStr = strings.Join(answers, ", ")
	}
	return respStr
}
