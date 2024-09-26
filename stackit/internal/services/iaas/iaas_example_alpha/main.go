package main

import (
	"context"
	"fmt"
	"os"

	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/iaasalpha"
)

func main() {
	projectId := "PROJECT_ID"

	// Create a new API client, that uses default authentication and configuration
	iaasClient, err := iaasalpha.NewAPIClient(
		config.WithRegion("eu01"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[IaaS API] Creating API client: %v\n", err)
		os.Exit(1)
	}

	// List the network for your project
	networks, err := iaasClient.ListNetworks(context.Background(), projectId).Execute()

	if err != nil {
		fmt.Fprintf(os.Stderr, "[IaaS API] Error when calling `ListNetworks`: %v\n", err)
	} else {
		fmt.Printf("[IaaS API] Number of networks: %v\n", len(*networks.Items))
	}
}
