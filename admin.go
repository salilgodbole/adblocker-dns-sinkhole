package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type AdminServer struct {
	logger    *QueryLogger
	blocklist *Blocklist
}

func NewAdminServer(logger *QueryLogger, blocklist *Blocklist) *AdminServer {
	return &AdminServer{logger: logger, blocklist: blocklist}
}

func (s *AdminServer) Start(port string) {
	http.HandleFunc("/", s.handleDashboard)
	http.HandleFunc("/api/logs", s.handleGetLogs)
	http.HandleFunc("/api/logs/clear", s.handleClearLogs)
	http.HandleFunc("/api/block", s.handleAddBlock)
	http.HandleFunc("/api/unblock", s.handleRemoveBlock)
	http.HandleFunc("/api/tracking/toggle", s.handleToggleTracking)
	http.HandleFunc("/api/tracking/state", s.handleTrackingState)
	http.HandleFunc("/api/blocking/toggle", s.handleToggleBlocking)
	http.HandleFunc("/api/blocking/state", s.handleBlockingState)

	fmt.Printf("Admin dashboard running at http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Println("Failed to start admin dashboard:", err)
	}
}

func (s *AdminServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(dashboardHTML))
}

func (s *AdminServer) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	logs := s.logger.GetLogs()
	json.NewEncoder(w).Encode(logs)
}

func (s *AdminServer) handleAddBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	domain := r.FormValue("domain")
	if domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	// Append to custom_blocklist.txt
	f, err := os.OpenFile(customBlocklistFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		http.Error(w, "Failed to open blocklist", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(domain + "\n"); err != nil {
		http.Error(w, "Failed to write to blocklist", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Domain added"))
}

func (s *AdminServer) handleRemoveBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	domain := r.FormValue("domain")
	if domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(customBlocklistFile)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "Failed to read blocklist", http.StatusInternalServerError)
		return
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != domain {
			newLines = append(newLines, line)
		}
	}

	err = os.WriteFile(customBlocklistFile, []byte(strings.Join(newLines, "\n")), 0644)
	if err != nil {
		http.Error(w, "Failed to write blocklist", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Domain removed"))
}

func (s *AdminServer) handleToggleTracking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	newState := !s.logger.IsTrackingEnabled()
	s.logger.SetTracking(newState)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"tracking": newState})
}

func (s *AdminServer) handleTrackingState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"tracking": s.logger.IsTrackingEnabled()})
}

func (s *AdminServer) handleToggleBlocking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	newState := !s.blocklist.IsBlockingEnabled()
	s.blocklist.SetBlocking(newState)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"blocking": newState})
}

func (s *AdminServer) handleBlockingState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"blocking": s.blocklist.IsBlockingEnabled()})
}

func (s *AdminServer) handleClearLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.logger.ClearLogs()
	w.WriteHeader(http.StatusOK)
}

const dashboardHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DNS Sinkhole Admin</title>
    <style>
        :root {
            --bg-color: #0f172a;
            --panel-bg: rgba(30, 41, 59, 0.7);
            --text-main: #f8fafc;
            --text-muted: #94a3b8;
            --border-color: rgba(255, 255, 255, 0.1);
            --accent: #3b82f6;
            --success: #10b981;
            --danger: #ef4444;
            --warning: #f59e0b;
        }
        
        body {
            background-color: var(--bg-color);
            color: var(--text-main);
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            margin: 0;
            padding: 2rem;
            min-height: 100vh;
            background-image: radial-gradient(circle at top right, rgba(59, 130, 246, 0.15), transparent 40%),
                              radial-gradient(circle at bottom left, rgba(16, 185, 129, 0.1), transparent 40%);
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
        }

        header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 2rem;
            border-bottom: 1px solid var(--border-color);
            padding-bottom: 1rem;
        }

        h1 {
            margin: 0;
            font-weight: 700;
            background: linear-gradient(to right, #60a5fa, #a78bfa);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .panels {
            display: grid;
            grid-template-columns: 300px 1fr;
            gap: 2rem;
        }

        .glass-panel {
            background: var(--panel-bg);
            backdrop-filter: blur(12px);
            -webkit-backdrop-filter: blur(12px);
            border: 1px solid var(--border-color);
            border-radius: 16px;
            padding: 1.5rem;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
        }

        h2 {
            margin-top: 0;
            font-size: 1.2rem;
            color: var(--text-muted);
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        .form-group {
            display: flex;
            flex-direction: column;
            gap: 0.75rem;
        }

        input[type="text"] {
            background: rgba(15, 23, 42, 0.6);
            border: 1px solid var(--border-color);
            color: white;
            padding: 0.75rem 1rem;
            border-radius: 8px;
            font-size: 1rem;
            outline: none;
            transition: all 0.2s;
        }

        input[type="text"]:focus {
            border-color: var(--accent);
            box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.25);
        }

        button {
            background: var(--accent);
            color: white;
            border: none;
            padding: 0.75rem 1rem;
            border-radius: 8px;
            font-size: 1rem;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s;
        }

        button:hover {
            background: #2563eb;
            transform: translateY(-1px);
        }

        button:active {
            transform: translateY(0);
        }

        #trackingBtn {
            background: rgba(255, 255, 255, 0.1);
            border: 1px solid var(--border-color);
        }

        #trackingBtn.active {
            background: rgba(16, 185, 129, 0.2);
            border-color: var(--success);
            color: var(--success);
        }

        #blockingBtn {
            background: rgba(16, 185, 129, 0.2);
            border: 1px solid var(--success);
            color: var(--success);
        }

        #blockingBtn.paused {
            background: rgba(239, 68, 68, 0.2);
            border-color: var(--danger);
            color: var(--danger);
        }

        table {
            width: 100%;
            border-collapse: collapse;
            font-size: 0.95rem;
        }

        th, td {
            padding: 1rem;
            text-align: left;
            border-bottom: 1px solid var(--border-color);
        }

        th {
            color: var(--text-muted);
            font-weight: 600;
        }

        tr:last-child td {
            border-bottom: none;
        }
        
        tr {
            transition: background-color 0.2s;
        }
        
        tr:hover {
            background-color: rgba(255, 255, 255, 0.02);
        }

        .status-badge {
            display: inline-block;
            padding: 0.25rem 0.75rem;
            border-radius: 9999px;
            font-size: 0.85rem;
            font-weight: 600;
        }

        .status-allowed {
            background: rgba(16, 185, 129, 0.15);
            color: var(--success);
        }

        .status-blocked {
            background: rgba(239, 68, 68, 0.15);
            color: var(--danger);
        }
        
        .status-custom {
            background: rgba(245, 158, 11, 0.15);
            color: var(--warning);
        }

        @media (max-width: 768px) {
            .panels {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div>
                <h1>DNS Sinkhole</h1>
                <div style="color: var(--text-muted); font-size: 0.9rem;">Live Traffic Monitor</div>
            </div>
            <div class="tracking-control" style="display: flex; gap: 1rem;">
                <button id="blockingBtn" onclick="toggleBlocking()">Pause Ad Blocking</button>
                <button id="trackingBtn" onclick="toggleTracking()">Enable Detailed Tracking</button>
            </div>
        </header>

        <div class="panels">
            <!-- Sidebar -->
            <div class="glass-panel" style="height: fit-content;">
                <h2>Manage Blocklist</h2>
                <p style="color: var(--text-muted); font-size: 0.9rem; margin-bottom: 1.5rem;">Instantly add or remove a domain from your custom blocklist.</p>
                <form id="blockForm" class="form-group">
                    <input type="text" id="domainInput" placeholder="e.g. ads.example.com" required>
                    <div style="display: flex; gap: 0.5rem;">
                        <button type="button" onclick="handleDomainAction('block')" style="flex: 1;">Block</button>
                        <button type="button" onclick="handleDomainAction('unblock')" style="flex: 1; background: var(--danger);">Unblock</button>
                    </div>
                </form>
                <div id="formMsg" style="margin-top: 1rem; font-size: 0.9rem; display: none;"></div>
            </div>

            <!-- Main Content -->
            <div class="glass-panel">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
                    <h2>Recent Queries</h2>
                    <div style="display: flex; gap: 1rem;">
                        <button onclick="clearLogs()" style="padding: 0.4rem 0.75rem; background: rgba(239, 68, 68, 0.2); border: 1px solid var(--danger); color: var(--danger); font-size: 0.9rem; margin-right: 0.5rem;">Clear Logs</button>
                        <select id="statusFilter" style="padding: 0.4rem 0.75rem; border-radius: 6px; background: rgba(15, 23, 42, 0.6); color: white; border: 1px solid var(--border-color); outline: none; font-size: 0.9rem;">
                            <option value="all">All Statuses</option>
                            <option value="allowed">Allowed Only</option>
                            <option value="blocked">Blocked Only</option>
                        </select>
                        <input type="text" id="ipFilter" placeholder="Filter by Client IP..." style="padding: 0.4rem 0.75rem; border-radius: 6px; width: 180px; font-size: 0.9rem; margin: 0;">
                    </div>
                </div>
                <div style="overflow-x: auto;">
                    <table>
                        <thead>
                            <tr>
                                <th>Time</th>
                                <th>Client IP</th>
                                <th>Type</th>
                                <th>Domain</th>
                                <th>Status</th>
                                <th>Response</th>
                            </tr>
                        </thead>
                        <tbody id="logsTable">
                            <tr>
                                <td colspan="6" style="text-align: center; color: var(--text-muted);">Loading...</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </div>

    <script>
        function formatTime(isoString) {
            const date = new Date(isoString);
            return date.toLocaleTimeString();
        }

        function getStatusBadge(status) {
            if (status === "Allowed") {
                return '<span class="status-badge status-allowed">Allowed</span>';
            } else if (status === "Blocked (Custom)") {
                return '<span class="status-badge status-custom">Custom Block</span>';
            }
            return '<span class="status-badge status-blocked">Blocked</span>';
        }

        async function fetchLogs() {
            try {
                const response = await fetch('/api/logs');
                const logs = await response.json();
                
                let filteredLogs = logs || [];
                
                const statusFilter = document.getElementById('statusFilter').value;
                if (statusFilter === 'allowed') {
                    filteredLogs = filteredLogs.filter(log => log.status === 'Allowed');
                } else if (statusFilter === 'blocked') {
                    filteredLogs = filteredLogs.filter(log => log.status !== 'Allowed');
                }

                const ipFilter = document.getElementById('ipFilter').value.trim();
                if (ipFilter) {
                    filteredLogs = filteredLogs.filter(log => log.client_ip.includes(ipFilter));
                }

                const tbody = document.getElementById('logsTable');
                if (filteredLogs.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="6" style="text-align: center; color: var(--text-muted);">No queries match filters.</td></tr>';
                    return;
                }

                tbody.innerHTML = filteredLogs.map(log => {
                    const domain = log.domain.endsWith('.') ? log.domain.slice(0, -1) : log.domain;
                    let inlineAction = '';
                    if (log.status === "Allowed") {
                        inlineAction = '<button onclick="handleInlineAction(event, \'block\', \'' + domain + '\')" style="margin-left: 0.5rem; padding: 0.2rem 0.5rem; font-size: 0.75rem; background: rgba(255,255,255,0.1); border: 1px solid var(--border-color);">Block</button>';
                    } else if (log.status === "Blocked (Custom)") {
                        inlineAction = '<button onclick="handleInlineAction(event, \'unblock\', \'' + domain + '\')" style="margin-left: 0.5rem; padding: 0.2rem 0.5rem; font-size: 0.75rem; background: rgba(239, 68, 68, 0.2); border: 1px solid var(--danger); color: var(--danger);">Unblock</button>';
                    }
                    return '<tr>' +
                        '<td style="color: var(--text-muted);">' + formatTime(log.timestamp) + '</td>' +
                        '<td>' + log.client_ip + '</td>' +
                        '<td style="color: var(--text-muted); font-size: 0.85rem;">' + (log.query_type || '-') + '</td>' +
                        '<td style="font-family: monospace; display: flex; justify-content: space-between; align-items: center; min-height: 28px;"><span>' + domain + '</span>' + inlineAction + '</td>' +
                        '<td>' + getStatusBadge(log.status) + '</td>' +
                        '<td style="font-family: monospace; font-size: 0.85rem; max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + (log.response || '') + '">' + (log.response || '-') + '</td>' +
                        '</tr>';
                }).join('');
            } catch (error) {
                console.error("Failed to fetch logs:", error);
            }
        }

        async function handleDomainAction(action) {
            const input = document.getElementById('domainInput');
            const domain = input.value.trim();
            const msgEl = document.getElementById('formMsg');
            
            if (!domain) return;

            const formData = new URLSearchParams();
            formData.append('domain', domain);

            try {
                const endpoint = action === 'block' ? '/api/block' : '/api/unblock';
                const res = await fetch(endpoint, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: formData
                });

                if (res.ok) {
                    input.value = '';
                    msgEl.textContent = action === 'block' ? "Domain blocked successfully!" : "Domain unblocked successfully!";
                    msgEl.style.color = "var(--success)";
                    msgEl.style.display = "block";
                    setTimeout(() => msgEl.style.display = "none", 3000);
                    fetchLogs();
                }
            } catch(e) {
                msgEl.textContent = "Error updating blocklist.";
                msgEl.style.color = "var(--danger)";
                msgEl.style.display = "block";
            }
        }

        async function handleInlineAction(e, action, domain) {
            e.preventDefault();
            const formData = new URLSearchParams();
            formData.append('domain', domain);
            const endpoint = action === 'block' ? '/api/block' : '/api/unblock';
            try {
                await fetch(endpoint, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                    body: formData
                });
                fetchLogs();
            } catch(error) {
                console.error("Inline action failed:", error);
            }
        }

        async function clearLogs() {
            if (!confirm("Are you sure you want to clear all logs?")) return;
            try {
                const res = await fetch('/api/logs/clear', { method: 'POST' });
                if (res.ok) fetchLogs();
            } catch(error) {
                console.error("Failed to clear logs:", error);
            }
        }

        let isTracking = false;
        let isBlocking = true;

        async function fetchTrackingState() {
            try {
                const res = await fetch('/api/tracking/state');
                const data = await res.json();
                isTracking = data.tracking;
                updateTrackingBtn();
            } catch (error) {
                console.error("Failed to fetch tracking state:", error);
            }
        }

        async function fetchBlockingState() {
            try {
                const res = await fetch('/api/blocking/state');
                const data = await res.json();
                isBlocking = data.blocking;
                updateBlockingBtn();
            } catch (error) {
                console.error("Failed to fetch blocking state:", error);
            }
        }

        async function toggleTracking() {
            try {
                const res = await fetch('/api/tracking/toggle', { method: 'POST' });
                const data = await res.json();
                isTracking = data.tracking;
                updateTrackingBtn();
            } catch (error) {
                console.error("Failed to toggle tracking:", error);
            }
        }

        async function toggleBlocking() {
            try {
                const res = await fetch('/api/blocking/toggle', { method: 'POST' });
                const data = await res.json();
                isBlocking = data.blocking;
                updateBlockingBtn();
            } catch (error) {
                console.error("Failed to toggle blocking:", error);
            }
        }

        function updateTrackingBtn() {
            const btn = document.getElementById('trackingBtn');
            if (isTracking) {
                btn.textContent = "Disable Detailed Tracking";
                btn.classList.add("active");
            } else {
                btn.textContent = "Enable Detailed Tracking";
                btn.classList.remove("active");
            }
        }

        function updateBlockingBtn() {
            const btn = document.getElementById('blockingBtn');
            if (isBlocking) {
                btn.textContent = "Pause Ad Blocking";
                btn.classList.remove("paused");
            } else {
                btn.textContent = "Resume Ad Blocking";
                btn.classList.add("paused");
            }
        }

        document.getElementById('statusFilter').addEventListener('change', fetchLogs);
        document.getElementById('ipFilter').addEventListener('input', fetchLogs);

        fetchTrackingState();
        fetchBlockingState();
        fetchLogs();
        setInterval(fetchLogs, 2000);
    </script>
</body>
</html>
`
