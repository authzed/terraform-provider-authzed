//go:build mage

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type OpenAPI mg.Namespace

// Update fetches the latest OpenAPI specification from the API
func (OpenAPI) Update() error {
	apiURL := "https://api.admin.stage.aws.authzed.net/openapi-spec"
	outputFile := "openapi-spec.yaml"

	fmt.Printf("Fetching latest OpenAPI spec from %s...\n", apiURL)

	client := &http.Client{}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers (User-Agent, Accept, etc.)
	req.Header.Set("User-Agent", "AuthZed-Terraform-Provider-Builder")
	req.Header.Set("Accept", "application/yaml, text/yaml, application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch OpenAPI spec: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch OpenAPI spec, status code: %d", resp.StatusCode)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write OpenAPI spec to file: %w", err)
	}

	fmt.Printf("OpenAPI spec successfully updated to %s\n", outputFile)

	// Check for changes if git is available
	if _, err := sh.Exec(nil, os.Stdout, os.Stderr, "git", "diff", "--quiet", outputFile); err != nil {
		fmt.Println("Changes detected in the OpenAPI spec.")
	} else {
		fmt.Println("No changes detected in the OpenAPI spec.")
	}

	return nil
}
