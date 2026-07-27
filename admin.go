package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type AdminServer struct {
	logger *QueryLogger
}

func NewAdminServer(logger *QueryLogger) *AdminServer {
	return &AdminServer{logger: logger}
}

func (s *AdminServer) Start(port string) {
	http.HandleFunc("/", s.handleDashboard)
	http.HandleFunc("/api/logs", s.handleGetLogs)
	http.HandleFunc("/api/block", s.handleAddBlock)
	http.HandleFunc("/api/tracking/toggle", s.handleToggleTracking)
	http.HandleFunc("/api/tracking/state", s.handleTrackingState)

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
            margin-left: 1rem;
        }

        #trackingBtn.active {
            background: rgba(16, 185, 129, 0.2);
            border-color: var(--success);
            color: var(--success);
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
            <div class="tracking-control">
                <button id="trackingBtn" onclick="toggleTracking()">Enable Detailed Tracking</button>
            </div>
        </header>

        <div class="panels">
            <!-- Sidebar -->
            <div class="glass-panel" style="height: fit-content;">
                <h2>Block Domain</h2>
                <p style="color: var(--text-muted); font-size: 0.9rem; margin-bottom: 1.5rem;">Instantly add a domain to your custom blocklist.</p>
                <form id="blockForm" class="form-group" onsubmit="addDomain(event)">
                    <input type="text" id="domainInput" placeholder="e.g. ads.example.com" required>
                    <button type="submit">Add to Blocklist</button>
                </form>
                <div id="formMsg" style="margin-top: 1rem; font-size: 0.9rem; display: none;"></div>
            </div>

            <!-- Main Content -->
            <div class="glass-panel">
                <h2>Recent Queries</h2>
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
                
                const tbody = document.getElementById('logsTable');
                if (!logs || logs.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="6" style="text-align: center; color: var(--text-muted);">No queries yet.</td></tr>';
                    return;
                }

                tbody.innerHTML = logs.map(log => {
                    const domain = log.domain.endsWith('.') ? log.domain.slice(0, -1) : log.domain;
                    return '<tr>' +
                        '<td style="color: var(--text-muted);">' + formatTime(log.timestamp) + '</td>' +
                        '<td>' + log.client_ip + '</td>' +
                        '<td style="color: var(--text-muted); font-size: 0.85rem;">' + (log.query_type || '-') + '</td>' +
                        '<td style="font-family: monospace;">' + domain + '</td>' +
                        '<td>' + getStatusBadge(log.status) + '</td>' +
                        '<td style="font-family: monospace; font-size: 0.85rem; max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="' + (log.response || '') + '">' + (log.response || '-') + '</td>' +
                        '</tr>';
                }).join('');
            } catch (error) {
                console.error("Failed to fetch logs:", error);
            }
        }

        async function addDomain(e) {
            e.preventDefault();
            const input = document.getElementById('domainInput');
            const domain = input.value.trim();
            const msgEl = document.getElementById('formMsg');
            
            if (!domain) return;

            const formData = new URLSearchParams();
            formData.append('domain', domain);

            try {
                const res = await fetch('/api/block', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: formData
                });

                if (res.ok) {
                    input.value = '';
                    msgEl.textContent = "Domain added successfully!";
                    msgEl.style.color = "var(--success)";
                    msgEl.style.display = "block";
                    setTimeout(() => msgEl.style.display = "none", 3000);
                    fetchLogs();
                }
            } catch(e) {
                msgEl.textContent = "Error adding domain.";
                msgEl.style.color = "var(--danger)";
                msgEl.style.display = "block";
            }
        }

        let isTracking = false;

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

        fetchTrackingState();
        fetchLogs();
        setInterval(fetchLogs, 2000);
    </script>
</body>
</html>
`
