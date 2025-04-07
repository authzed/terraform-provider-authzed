//go:build ignore

package main

import (
	"fmt"
	"terraform-provider-cloud-api/internal/client"
)

func main() {
	// Create client with the same config as Terraform would use
	cloudClient := client.NewCloudClient(&client.CloudClientConfig{
		Host:       "https://api.admin.stage.aws.authzed.net", // Without trailing slash
		Token:      "04i8YhUx2QpEg1weH3do7PFbKWAGvnR9",
		APIVersion: "25r1",
	})

	fmt.Println("\n=== LISTING PERMISSION SYSTEMS ===")
	psList, err := cloudClient.ListPermissionSystems()
	if err != nil {
		fmt.Printf("Error listing permission systems: %s\n", err)
		return
	}

	fmt.Printf("Found %d permission systems:\n", len(psList))
	for i, ps := range psList {
		fmt.Printf("%d. ID: %s, Name: %s\n", i+1, ps.ID, ps.Name)
	}
}
