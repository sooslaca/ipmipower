package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
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

func main() {
	// Define command line flags
	var (
		host     = flag.String("host", "", "BMC IP address (default is IPMI_HOST environment variable)")
		username = flag.String("username", "", "BMC username (default is IPMI_USERNAME environment variable)")
		password = flag.String("password", "", "BMC password (default is IPMI_PASSWORD environment variable)")
		port     = flag.Int("port", 623, "BMC port")
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

	// 2. Create the Client
	client, err := ipmi.NewClient(finalHost, *port, finalUsername, finalPassword)
	if err != nil {
		log.Fatalf("Failed to create IPMI client: %v", err)
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
		log.Fatalf("Failed to establish session: %v", err)
	}
	defer client.Close(ctx)

	fmt.Printf("Connected to BMC at %s\n", finalHost)

	// 5. Check Current Power Status
	status, err := client.GetChassisStatus(ctx)
	if err != nil {
		log.Fatalf("Failed to get chassis status: %v", err)
	}

	// 6. Conditional Logic
	// status.PowerIsOn is a boolean helper provided by the library
	if status.PowerIsOn {
		fmt.Println("The server is already powered ON. No action taken.")
		return
	}

	fmt.Println("Server is currently OFF. Sending Power Up command...")

	// 7. Send Power Up Command
	if _, err := client.ChassisControl(ctx, ipmi.ChassisControlPowerUp); err != nil {
		log.Fatalf("Failed to send Power Up command: %v", err)
	}

	fmt.Println("Success: Power On command sent.")
}
