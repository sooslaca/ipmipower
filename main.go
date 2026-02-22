package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bougou/go-ipmi"
)

// getEnvOrDefault returns the value of the environment variable with the given name,
// or the default value if the environment variable is not set.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseMAC converts a MAC address string in various formats to a byte slice
func parseMAC(macStr string) ([]byte, error) {
	// Replace hyphens with colons for consistent parsing
	macStr = strings.ReplaceAll(macStr, "-", ":")

	hwAddr, err := net.ParseMAC(macStr)
	if err != nil {
		return nil, err
	}

	return hwAddr, nil
}

// isValidWOLPacket checks if the received packet is a valid Wake-on-LAN magic packet
func isValidWOLPacket(data []byte, targetMAC []byte) bool {
	if len(data) != 102 { // WoL packet is 6 bytes of 0xFF + 16 repetitions of MAC address (6*17 = 102)
		return false
	}

	// Check for 6 bytes of 0xFF at the beginning
	for i := 0; i < 6; i++ {
		if data[i] != 0xFF {
			return false
		}
	}

	// Check for 16 repetitions of the target MAC address
	for i := 0; i < 16; i++ {
		for j := 0; j < 6; j++ {
			if data[6+i*6+j] != targetMAC[j] {
				return false
			}
		}
	}

	return true
}

// sendIPMIPowerOn sends the IPMI power on command using the same logic as the original code
func sendIPMIPowerOn(host string, username string, password string, port int) error {
	// 2. Create the Client
	client, err := ipmi.NewClient(host, port, username, password)
	if err != nil {
		return fmt.Errorf("failed to create IPMI client: %v", err)
	}

	// 3. Configure for Remote Access
	// "lanplus" is the standard for RMCP+ (IPMI 2.0) remote management.
	client.WithMaxPrivilegeLevel(ipmi.PrivilegeLevelOperator)
	client.WithInterface("lanplus")

	// You can also set a timeout for the network operations
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 4. Connect and Authenticate
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to establish session: %v", err)
	}
	defer client.Close(ctx)

	fmt.Printf("Connected to BMC at %s\n", host)

	// 5. Check Current Power Status
	status, err := client.GetChassisStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chassis status: %v", err)
	}

	// 6. Conditional Logic
	// status.PowerIsOn is a boolean helper provided by the library
	if status.PowerIsOn {
		fmt.Println("The server is already powered ON. No action taken.")
		return nil
	}

	fmt.Println("Server is currently OFF. Sending Power Up command...")

	// 7. Send Power Up Command
	if _, err := client.ChassisControl(ctx, ipmi.ChassisControlPowerUp); err != nil {
		return fmt.Errorf("failed to send Power Up command: %v", err)
	}

	fmt.Println("Success: Power On command sent.")
	return nil
}

// getPowerStatus gets the current power status of the IPMI host
func getPowerStatus(host string, username string, password string, port int) (bool, error) {
	// 2. Create the Client
	client, err := ipmi.NewClient(host, port, username, password)
	if err != nil {
		return false, fmt.Errorf("failed to create IPMI client: %v", err)
	}

	// 3. Configure for Remote Access
	// "lanplus" is the standard for RMCP+ (IPMI 2.0) remote management.
	client.WithMaxPrivilegeLevel(ipmi.PrivilegeLevelOperator)
	client.WithInterface("lanplus")

	// You can also set a timeout for the network operations
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 4. Connect and Authenticate
	if err := client.Connect(ctx); err != nil {
		return false, fmt.Errorf("failed to establish session: %v", err)
	}
	defer client.Close(ctx)

	// 5. Check Current Power Status
	status, err := client.GetChassisStatus(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get chassis status: %v", err)
	}

	return status.PowerIsOn, nil
}

// WebPageData represents the data to be passed to the HTML template
type WebPageData struct {
	Host      string
	Status    string
	IsPowered bool
}

// webHandler handles the main web page
func webHandler(w http.ResponseWriter, r *http.Request) {
	// Get configuration from environment variables
	host := getEnvOrDefault("IPMI_HOST", "192.168.0.1")
	username := getEnvOrDefault("IPMI_USERNAME", "admin")
	password := getEnvOrDefault("IPMI_PASSWORD", "admin")
	port := 623 // Default IPMI port

	// Get current power status
	isPowered, err := getPowerStatus(host, username, password, port)
	var statusStr string

	// Handle error by setting status to "Unknown" and isPowered to false
	if err != nil {
		log.Printf("Error getting power status: %v", err)
		isPowered = false // Default to powered off when there's an error
		statusStr = "Unknown"
	} else {
		statusStr = getStatusString(isPowered)
	}

	// Prepare data for template
	data := WebPageData{
		Host:      host,
		Status:    statusStr,
		IsPowered: isPowered,
	}

	// HTML template
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>IPMI Power Control</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 600px;
            margin: 50px auto;
            padding: 20px;
            text-align: center;
        }
        .status {
            font-size: 24px;
            margin: 20px 0;
            padding: 15px;
            border-radius: 5px;
        }
        .on {
            background-color: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        .off {
            background-color: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }
        .unknown {
            background-color: #fff3cd;
            color: #856404;
            border: 1px solid #ffeaa7;
        }
        button {
            background-color: #007bff;
            color: white;
            border: none;
            padding: 12px 24px;
            font-size: 16px;
            border-radius: 4px;
            cursor: pointer;
            margin: 10px;
        }
        button:hover {
            background-color: #0056b3;
        }
        button:disabled {
            background-color: #6c757d;
            cursor: not-allowed;
        }
        .refresh-btn {
            background-color: #28a745;
        }
        .refresh-btn:hover {
            background-color: #1e7e34;
        }
    </style>
</head>
<body>
    <h1>IPMI Power Control</h1>
    <div class="status {{if eq .Status "ON"}}on{{else if eq .Status "Unknown"}}unknown{{else}}off{{end}}">
        <h2>Host: {{.Host}}</h2>
        <p>Power Status: <strong>{{.Status}}</strong></p>
    </div>

    {{if not .IsPowered}}
        <form method="POST" action="/poweron" style="display: inline;">
            <button type="submit">Power ON</button>
        </form>
    {{else}}
        <button disabled>Power ON (already on)</button>
    {{end}}

    <form method="GET" action="/" style="display: inline;">
        <button type="submit" class="refresh-btn">Refresh Status</button>
    </form>

    <div style="margin-top: 30px; font-size: 14px; color: #666;">
        <p>Current time: {{.Time}}</p>
    </div>

    {{if eq .Status "Unknown"}}
    <script>
        // Refresh the page every 1 second when status is Unknown
        setTimeout(function() {
            window.location.reload();
        }, 1000);
    </script>
    {{end}}
</body>
</html>
`

	// Parse and execute template
	t, err := template.New("webpage").Parse(tmpl)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing template: %v", err), http.StatusInternalServerError)
		return
	}

	// Add current time to data
	dataWithTime := struct {
		WebPageData
		Time string
	}{
		WebPageData: data,
		Time:        time.Now().Format("2006-01-02 15:04:05"),
	}

	if err := t.Execute(w, dataWithTime); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

// powerOnHandler handles the power on command
func powerOnHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get configuration from environment variables
	host := getEnvOrDefault("IPMI_HOST", "192.168.0.1")
	username := getEnvOrDefault("IPMI_USERNAME", "admin")
	password := getEnvOrDefault("IPMI_PASSWORD", "admin")
	port := 623 // Default IPMI port

	// Send power on command
	err := sendIPMIPowerOn(host, username, password, port)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error powering on: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect back to main page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// getStatusString returns a string representation of the power status
func getStatusString(isPowered bool) string {
	if isPowered {
		return "ON"
	}
	return "OFF"
}

func main() {
	// Define command line flags
	var (
		host     = flag.String("host", "", "BMC IP address (default is IPMI_HOST environment variable)")
		username = flag.String("username", "", "BMC username (default is IPMI_USERNAME environment variable)")
		password = flag.String("password", "", "BMC password (default is IPMI_PASSWORD environment variable)")
		port     = flag.Int("port", 623, "BMC port")
		wolPort  = flag.Int("wol-port", 9, "WoL port to listen on")
		mac      = flag.String("mac", "00:11:22:33:44:55", "Target MAC address to listen for (format: 00-11-22-33-44-55 or 00:11:22:33:44:55)")
		mode     = flag.String("mode", "wol", "Operation mode: 'wol' for Wake-on-LAN server, 'direct' for direct IPMI command")
		webPort  = flag.Int("web-port", 80, "Web server port")
	)

	// Parse command line flags
	flag.Parse()

	// Get values from flags, falling back to environment variables if flags are not provided
	finalHost := *host
	if finalHost == "" {
		finalHost = getEnvOrDefault("IPMI_HOST", "192.168.0.1")
	}

	finalUsername := *username
	if finalUsername == "" {
		finalUsername = getEnvOrDefault("IPMI_USERNAME", "admin")
	}

	finalPassword := *password
	if finalPassword == "" {
		finalPassword = getEnvOrDefault("IPMI_PASSWORD", "admin")
	}

	// Get MAC address from flag or environment variable
	finalMAC := *mac
	if envMAC := getEnvOrDefault("WOL_MAC", ""); envMAC != "" {
		finalMAC = envMAC
	}

	// Parse the target MAC address
	targetMAC, err := parseMAC(finalMAC)
	if err != nil {
		log.Fatalf("Invalid MAC address format: %v", err)
	}

	// Determine operation mode
	if *mode == "direct" {
		// Execute direct IPMI command (original functionality)
		if err := sendIPMIPowerOn(finalHost, finalUsername, finalPassword, *port); err != nil {
			log.Fatalf("Error executing IPMI command: %v", err)
		}
		return
	}

	// Set up web server routes
	http.HandleFunc("/", webHandler)
	http.HandleFunc("/poweron", powerOnHandler)

	// Start web server in a separate goroutine if not in WoL-only mode
	go func() {
		fmt.Printf("Starting web server on port %d\n", *webPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *webPort), nil); err != nil {
			log.Fatalf("Web server error: %v", err)
		}
	}()

	// WoL mode: Listen for magic packets
	fmt.Printf("Starting WoL listener on port %d, waiting for MAC: %s\n", *wolPort, finalMAC)

	// Create UDP listener for WoL packets
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", *wolPort))
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on UDP port %d: %v", *wolPort, err)
	}
	defer conn.Close()

	fmt.Printf("Listening for WoL packets on port %d...\n", *wolPort)

	buffer := make([]byte, 1024) // Buffer to store incoming data
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading UDP message: %v", err)
			continue
		}

		// Check if the received packet is a valid WoL magic packet for our target MAC
		if n >= 102 && isValidWOLPacket(buffer[:102], targetMAC) {
			fmt.Printf("Received valid WoL packet from %s for MAC %s\n", clientAddr, finalMAC)

			// Execute IPMI power on command
			if err := sendIPMIPowerOn(finalHost, finalUsername, finalPassword, *port); err != nil {
				log.Printf("Error executing IPMI command: %v", err)
			}
		}
	}
}
