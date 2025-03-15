/**
 * WalletConnect Test App
 * Main JavaScript file
 */

// Helper function to add logs to the log container
function addLog(message, container) {
    const logContainer = container || document.getElementById('logs');
    if (!logContainer) return;
    
    const timestamp = new Date().toLocaleTimeString();
    const logEntry = document.createElement('p');
    logEntry.className = 'log-entry';
    logEntry.textContent = `[${timestamp}] ${message}`;
    
    logContainer.appendChild(logEntry);
    logContainer.scrollTop = logContainer.scrollHeight;
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

// Add click-to-copy functionality to address and signature elements
document.addEventListener('DOMContentLoaded', function() {
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
    
    // Log initialization
    const logContainer = document.getElementById('logs');
    if (logContainer) {
        addLog('Page initialized', logContainer);
        addLog('WalletConnect Test App ready', logContainer);
    }
});

// Export functions for use in inline scripts
window.wcTestApp = {
    addLog: addLog,
    copyToClipboard: copyToClipboard
};