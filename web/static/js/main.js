/**
 * WalletConnect Test App
 * Main JavaScript file
 */

// Helper function to add logs to the log container
function addLog(message, type = 'info', container) {
    const logContainer = container || document.getElementById('logs');
    if (!logContainer) return;
    
    // Skip verbose logs if verbose logging is disabled and no container is specified
    if (type === 'verbose' && !window.wcTestApp.verboseLogging && !container) return;
    
    const timestamp = new Date().toLocaleTimeString() + '.' + String(new Date().getMilliseconds()).padStart(3, '0');
    const logEntry = document.createElement('p');
    logEntry.className = `log-entry log-${type}`;
    logEntry.textContent = `[${timestamp}] ${message}`;
    
    logContainer.appendChild(logEntry);
    logContainer.scrollTop = logContainer.scrollHeight;
    
    // Also log to console for debugging
    if (type === 'error') {
        console.error(`[${timestamp}] ${message}`);
    } else if (type === 'verbose') {
        console.debug(`[${timestamp}] ${message}`);
    } else {
        console.log(`[${timestamp}] ${message}`);
    }
}

// Helper function to copy text to clipboard
function copyToClipboard(text) {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    document.body.appendChild(textarea);
    textarea.select();
    
    try {
        const successful = document.execCommand('copy');
        return successful;
    } catch (err) {
        console.error('Failed to copy text: ', err);
        return false;
    } finally {
        document.body.removeChild(textarea);
    }
}

// Helper function to format JSON
function formatJSON(jsonStr) {
    try {
        const obj = JSON.parse(jsonStr);
        return JSON.stringify(obj, null, 2);
    } catch (e) {
        return jsonStr;
    }
}

// Helper function to truncate long strings
function truncateString(str, maxLength = 100) {
    if (!str) return '';
    if (str.length <= maxLength) return str;
    return str.substring(0, maxLength) + '...';
}

// WebSocket debugging utilities
const wsDebugger = {
    // Track all WebSocket connections
    connections: {},
    
    // Intercept WebSocket constructor
    setupInterceptor: function() {
        const OriginalWebSocket = window.WebSocket;
        const self = this;
        
        window.WebSocket = function(url, protocols) {
            // Create the WebSocket
            const ws = new OriginalWebSocket(url, protocols);
            const wsId = Date.now().toString();
            
            // Store the connection
            self.connections[wsId] = {
                url: url,
                protocols: protocols,
                ws: ws,
                status: 'connecting',
                messagesSent: 0,
                messagesReceived: 0,
                createdAt: new Date(),
                lastActivity: new Date()
            };
            
            // Log the connection
            addLog(`WebSocket connection created to ${url}`, 'verbose');
            addLog(`WebSocket protocol: ${protocols || 'none'}`, 'verbose');
            
            // Intercept events
            ws.addEventListener('open', function() {
                self.connections[wsId].status = 'open';
                self.connections[wsId].lastActivity = new Date();
                addLog(`WebSocket connection opened to ${url}`, 'verbose');
            });
            
            ws.addEventListener('close', function(event) {
                self.connections[wsId].status = 'closed';
                self.connections[wsId].lastActivity = new Date();
                self.connections[wsId].closeCode = event.code;
                self.connections[wsId].closeReason = event.reason;
                addLog(`WebSocket connection closed: Code ${event.code}, Reason: ${event.reason || 'none'}`, 'verbose');
            });
            
            ws.addEventListener('error', function(error) {
                self.connections[wsId].status = 'error';
                self.connections[wsId].lastActivity = new Date();
                self.connections[wsId].lastError = error;
                addLog(`WebSocket error: ${error}`, 'error');
            });
            
            // Intercept message events
            ws.addEventListener('message', function(event) {
                self.connections[wsId].messagesReceived++;
                self.connections[wsId].lastActivity = new Date();
                self.connections[wsId].lastMessage = event.data;
                
                addLog(`WebSocket received message (${event.data.length} bytes)`, 'verbose');
                try {
                    // Try to parse as JSON
                    const data = JSON.parse(event.data);
                    addLog(`Received message: ${truncateString(JSON.stringify(data, null, 2))}`, 'verbose');
                } catch (e) {
                    // Not JSON, just log the first part
                    addLog(`Received raw message: ${truncateString(event.data)}`, 'verbose');
                }
            });
            
            // Intercept send method
            const originalSend = ws.send;
            ws.send = function(data) {
                self.connections[wsId].messagesSent++;
                self.connections[wsId].lastActivity = new Date();
                self.connections[wsId].lastSentMessage = data;
                
                addLog(`WebSocket sending message (${data.length} bytes)`, 'verbose');
                try {
                    // Try to parse as JSON
                    const jsonData = JSON.parse(data);
                    addLog(`Sent message: ${truncateString(JSON.stringify(jsonData, null, 2))}`, 'verbose');
                } catch (e) {
                    // Not JSON, just log the first part
                    addLog(`Sent raw message: ${truncateString(data)}`, 'verbose');
                }
                
                return originalSend.apply(this, arguments);
            };
            
            return ws;
        };
    },
    
    // Get stats about all connections
    getStats: function() {
        const stats = {
            totalConnections: Object.keys(this.connections).length,
            openConnections: 0,
            totalMessagesSent: 0,
            totalMessagesReceived: 0,
            connections: []
        };
        
        for (const id in this.connections) {
            const conn = this.connections[id];
            if (conn.status === 'open') stats.openConnections++;
            stats.totalMessagesSent += conn.messagesSent;
            stats.totalMessagesReceived += conn.messagesReceived;
            
            stats.connections.push({
                url: conn.url,
                status: conn.status,
                messagesSent: conn.messagesSent,
                messagesReceived: conn.messagesReceived,
                createdAt: conn.createdAt,
                lastActivity: conn.lastActivity
            });
        }
        
        return stats;
    },
    
    // Log stats about all connections
    logStats: function() {
        const stats = this.getStats();
        addLog(`WebSocket Stats: ${stats.openConnections}/${stats.totalConnections} connections open`, 'verbose');
        addLog(`Total messages: ${stats.totalMessagesSent} sent, ${stats.totalMessagesReceived} received`, 'verbose');
        
        stats.connections.forEach((conn, i) => {
            const age = Math.round((new Date() - conn.createdAt) / 1000);
            const lastActivity = Math.round((new Date() - conn.lastActivity) / 1000);
            
            addLog(`Connection ${i+1}: ${conn.url} (${conn.status})`, 'verbose');
            addLog(`  Age: ${age}s, Last activity: ${lastActivity}s ago`, 'verbose');
            addLog(`  Messages: ${conn.messagesSent} sent, ${conn.messagesReceived} received`, 'verbose');
        });
    }
};

// Add click-to-copy functionality to address and signature elements
document.addEventListener('DOMContentLoaded', function() {
    // Initialize WebSocket debugging
    wsDebugger.setupInterceptor();
    
    // Add copy functionality to address elements
    const addressElements = document.querySelectorAll('.address');
    addressElements.forEach(element => {
        element.title = 'Click to copy';
        element.style.cursor = 'pointer';
        
        element.addEventListener('click', function() {
            const text = this.textContent;
            const success = copyToClipboard(text);
            
            if (success) {
                const originalText = this.textContent;
                this.textContent = 'Copied!';
                
                setTimeout(() => {
                    this.textContent = originalText;
                }, 1500);
            }
        });
    });
    
    // Add copy functionality to signature elements
    const signatureElements = document.querySelectorAll('.signature');
    signatureElements.forEach(element => {
        element.title = 'Click to copy';
        element.style.cursor = 'pointer';
        
        element.addEventListener('click', function() {
            const text = this.textContent;
            const success = copyToClipboard(text);
            
            if (success) {
                const originalText = this.textContent;
                this.textContent = 'Copied!';
                
                setTimeout(() => {
                    this.textContent = originalText;
                }, 1500);
            }
        });
    });
    
    // Set up toggle verbose buttons
    const toggleVerboseButtons = document.querySelectorAll('#toggle-verbose');
    toggleVerboseButtons.forEach(button => {
        button.addEventListener('click', function() {
            window.wcTestApp.verboseLogging = !window.wcTestApp.verboseLogging;
            const toggleButtons = document.querySelectorAll('#toggle-verbose');
            toggleButtons.forEach(btn => {
                btn.textContent = window.wcTestApp.verboseLogging ? 'Hide Detailed Logs' : 'Show Detailed Logs';
            });
        });
    });
    
    // Set up clear logs buttons
    const clearLogsButtons = document.querySelectorAll('#clear-logs');
    clearLogsButtons.forEach(button => {
        button.addEventListener('click', function() {
            const logContainer = document.getElementById('logs');
            if (logContainer) {
                logContainer.innerHTML = '';
                addLog('Logs cleared', 'system');
            }
        });
    });
    
    // Log initialization
    const logContainer = document.getElementById('logs');
    if (logContainer) {
        addLog('Page initialized', 'info', logContainer);
        addLog('WalletConnect Test App ready', 'info', logContainer);
        addLog('WebSocket debugging enabled', 'verbose', logContainer);
        
        // Periodically log WebSocket stats (every 30 seconds)
        setInterval(() => {
            if (window.wcTestApp.verboseLogging) {
                wsDebugger.logStats();
            }
        }, 30000);
    }
});

// Export functions for use in inline scripts
window.wcTestApp = {
    addLog: addLog,
    copyToClipboard: copyToClipboard,
    formatJSON: formatJSON,
    truncateString: truncateString,
    wsDebugger: wsDebugger,
    verboseLogging: false
};