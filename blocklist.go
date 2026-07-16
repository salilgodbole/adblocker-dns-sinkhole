package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const blocklistURL = "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts"
const customBlocklistFile = "custom_blocklist.txt"

type Blocklist struct {
	mu            sync.RWMutex
	domains       map[string]struct{}
	customDomains map[string]struct{}
}

func NewBlocklist() *Blocklist {
	return &Blocklist{
		domains:       make(map[string]struct{}),
		customDomains: make(map[string]struct{}),
	}
}

func (b *Blocklist) Load() error {
	fmt.Println("Downloading community blocklist...")
	resp, err := http.Get(blocklistURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	newDomains := make(map[string]struct{})

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
				newDomains[domain] = struct{}{}
			}
		}
	}

	b.mu.Lock()
	b.domains = newDomains
	b.mu.Unlock()

	fmt.Printf("Loaded %d community blocked domains\n", len(b.domains))
	return scanner.Err()
}

func (b *Blocklist) loadCustomList() {
	file, err := os.Open(customBlocklistFile)
	if err != nil {
		// It's okay if the file doesn't exist yet
		if os.IsNotExist(err) {
			b.mu.Lock()
			b.customDomains = make(map[string]struct{})
			b.mu.Unlock()
		}
		return
	}
	defer file.Close()

	newCustomDomains := make(map[string]struct{})
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Expect just the domain name per line
		newCustomDomains[line] = struct{}{}
	}

	b.mu.Lock()
	// Only print if the count changed to avoid spamming the console
	if len(b.customDomains) != len(newCustomDomains) {
		fmt.Printf("Loaded %d custom domains from %s\n", len(newCustomDomains), customBlocklistFile)
	}
	b.customDomains = newCustomDomains
	b.mu.Unlock()
}

// WatchCustomBlocklist polls the custom blocklist file every 5 seconds and updates the map instantly
func (b *Blocklist) WatchCustomBlocklist() {
	// Create an empty file if it doesn't exist
	if _, err := os.Stat(customBlocklistFile); os.IsNotExist(err) {
		os.WriteFile(customBlocklistFile, []byte("# Add your custom domains here, one per line\n"), 0644)
	}

	// Load initially
	b.loadCustomList()

	// Poll every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		b.loadCustomList()
	}
}

func (b *Blocklist) IsBlocked(domain string) (bool, LogStatus) {
	domain = strings.TrimSuffix(domain, ".")
	
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check custom first
	if _, exists := b.customDomains[domain]; exists {
		return true, StatusBlockedCustom
	}

	// Then check community
	if _, exists := b.domains[domain]; exists {
		return true, StatusBlockedDefault
	}
	
	return false, StatusAllowed
}
