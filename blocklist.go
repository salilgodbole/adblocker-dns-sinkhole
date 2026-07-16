package main

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

const blocklistURL = "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts"

type Blocklist struct {
	mu      sync.RWMutex
	domains map[string]struct{}
}

func NewBlocklist() *Blocklist {
	return &Blocklist{
		domains: make(map[string]struct{}),
	}
}

func (b *Blocklist) Load() error {
	fmt.Println("Downloading blocklist...")
	resp, err := http.Get(blocklistURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b.mu.Lock()
	defer b.mu.Unlock()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 && (parts[0] == "0.0.0.0" || parts[0] == "127.0.0.1") {
			domain := parts[1]
			if domain != "0.0.0.0" && domain != "127.0.0.1" && domain != "localhost" && domain != "broadcasthost" {
				b.domains[domain] = struct{}{}
			}
		}
	}

	fmt.Printf("Loaded %d blocked domains\n", len(b.domains))
	return scanner.Err()
}

func (b *Blocklist) IsBlocked(domain string) bool {
	// DNS queries usually have a trailing dot, we need to remove it for matching
	domain = strings.TrimSuffix(domain, ".")
	
	b.mu.RLock()
	defer b.mu.RUnlock()

	_, exists := b.domains[domain]
	return exists
}
