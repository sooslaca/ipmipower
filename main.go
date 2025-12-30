package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
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

	fmt.Printf("Target MAC address: %s\n", finalMAC)

	// Determine operation mode
	if *mode == "direct" {
		// Execute direct IPMI command (original functionality)
		if err := sendIPMIPowerOn(finalHost, finalUsername, finalPassword, *port); err != nil {
			log.Fatalf("Error executing IPMI command: %v", err)
		}
		return
	}

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
