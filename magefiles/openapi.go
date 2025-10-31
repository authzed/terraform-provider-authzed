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

// Update fetches the latest OpenAPI specification from the private GitHub repository
func (OpenAPI) Update() error {
	apiURL := "https://raw.githubusercontent.com/authzed/internal/main/cloud-api/internal/specs/25r1.yaml"
	outputFile := "openapi-spec.yaml"

	// Get GitHub token from environment
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}

	fmt.Printf("Fetching latest OpenAPI spec from GitHub repository...\n")

	client := &http.Client{}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers (User-Agent, Accept, Authorization)
	req.Header.Set("User-Agent", "AuthZed-Terraform-Provider-Builder")
	req.Header.Set("Accept", "application/yaml, text/yaml, application/json")
	req.Header.Set("Authorization", "token "+githubToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch OpenAPI spec: %w", err)
	}
	defer resp.Body.Close()

	// Check response status with detailed error messages
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed (status 401). Please check GITHUB_TOKEN")
		case http.StatusForbidden:
			return fmt.Errorf("access forbidden (status 403). Token may lack required permissions")
		case http.StatusNotFound:
			return fmt.Errorf("file not found (status 404). Check repository and file path")
		default:
			return fmt.Errorf("failed to fetch OpenAPI spec, status code: %d", resp.StatusCode)
		}
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
