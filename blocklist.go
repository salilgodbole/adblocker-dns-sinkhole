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
const customAllowlistFile = "custom_allowlist.txt"

type Blocklist struct {
	mu              sync.RWMutex
	domains         map[string]struct{}
	customDomains   map[string]struct{}
	allowDomains    map[string]struct{}
	blockingEnabled bool
}

func NewBlocklist() *Blocklist {
	return &Blocklist{
		domains:         make(map[string]struct{}),
		customDomains:   make(map[string]struct{}),
		allowDomains:    make(map[string]struct{}),
		blockingEnabled: true,
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
		fmt.Printf("Loaded %d custom blocked domains from %s\n", len(newCustomDomains), customBlocklistFile)
	}
	b.customDomains = newCustomDomains
	b.mu.Unlock()
}

func (b *Blocklist) loadAllowList() {
	file, err := os.Open(customAllowlistFile)
	if err != nil {
		if os.IsNotExist(err) {
			b.mu.Lock()
			b.allowDomains = make(map[string]struct{})
			b.mu.Unlock()
		}
		return
	}
	defer file.Close()

	newAllowDomains := make(map[string]struct{})
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		newAllowDomains[line] = struct{}{}
	}

	b.mu.Lock()
	if len(b.allowDomains) != len(newAllowDomains) {
		fmt.Printf("Loaded %d custom allowed domains from %s\n", len(newAllowDomains), customAllowlistFile)
	}
	b.allowDomains = newAllowDomains
	b.mu.Unlock()
}

// WatchCustomBlocklist polls the custom lists every 5 seconds and updates the maps instantly
func (b *Blocklist) WatchCustomBlocklist() {
	// Create empty files if they don't exist
	if _, err := os.Stat(customBlocklistFile); os.IsNotExist(err) {
		os.WriteFile(customBlocklistFile, []byte("# Add your custom domains here, one per line\n"), 0644)
	}
	if _, err := os.Stat(customAllowlistFile); os.IsNotExist(err) {
		os.WriteFile(customAllowlistFile, []byte("# Add domains to ALWAYS ALLOW here, one per line\n"), 0644)
	}

	// Load initially
	b.loadCustomList()
	b.loadAllowList()

	// Poll every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		b.loadCustomList()
		b.loadAllowList()
	}
}

func (b *Blocklist) SetBlocking(enabled bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.blockingEnabled = enabled
}

func (b *Blocklist) IsBlockingEnabled() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.blockingEnabled
}

func (b *Blocklist) IsBlocked(domain string) (bool, LogStatus) {
	domain = strings.TrimSuffix(domain, ".")
	
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.blockingEnabled {
		return false, StatusAllowed
	}

	// Check allowlist first
	if _, exists := b.allowDomains[domain]; exists {
		return false, StatusAllowedCustom
	}

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
