package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bougou/go-ipmi"
)

func main() {
	// 1. Configuration - Update these with your specific details
	host := "192.168.0.1" // Your BMC IP address
	port := 623
	username := "admin"
	password := "admin"

	// 2. Create the Client
	client, err := ipmi.NewClient(host, port, username, password)
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

	fmt.Printf("Connected to BMC at %s\n", host)

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
