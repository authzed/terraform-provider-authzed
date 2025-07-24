package helpers

import (
	"fmt"
	"log"
	"strings"
	"time"

	"terraform-provider-authzed/internal/client"
)

// CleanupReport represents the results of a cleanup verification
type CleanupReport struct {
	PermissionSystemID string             `json:"permission_system_id"`
	Timestamp          time.Time          `json:"timestamp"`
	ResourceCounts     map[string]int     `json:"resource_counts"`
	OrphanedResources  []OrphanedResource `json:"orphaned_resources"`
	CleanupActions     []CleanupAction    `json:"cleanup_actions"`
	Success            bool               `json:"success"`
	ErrorMessages      []string           `json:"error_messages"`
}

// OrphanedResource represents a resource that should have been cleaned up
type OrphanedResource struct {
	Type       string    `json:"type"`
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	TestPrefix string    `json:"test_prefix"`
}

// CleanupAction represents an action taken during cleanup
type CleanupAction struct {
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// CleanupVerifier handles verification and cleanup of test resources
type CleanupVerifier struct {
	client             *client.CloudClient
	permissionSystemID string
	testPrefixes       []string
}

// NewCleanupVerifier creates a new cleanup verifier instance
func NewCleanupVerifier(permissionSystemID string, testPrefixes []string) (*CleanupVerifier, error) {
	clientConfig := &client.CloudClientConfig{
		Host:       GetTestHost(),
		Token:      GetTestToken(),
		APIVersion: GetTestAPIVersion(),
	}

	testClient := client.NewCloudClient(clientConfig)

	return &CleanupVerifier{
		client:             testClient,
		permissionSystemID: permissionSystemID,
		testPrefixes:       testPrefixes,
	}, nil
}

// VerifyCleanup performs comprehensive cleanup verification
func (cv *CleanupVerifier) VerifyCleanup() (*CleanupReport, error) {
	report := &CleanupReport{
		PermissionSystemID: cv.permissionSystemID,
		Timestamp:          time.Now(),
		ResourceCounts:     make(map[string]int),
		OrphanedResources:  []OrphanedResource{},
		CleanupActions:     []CleanupAction{},
		Success:            true,
		ErrorMessages:      []string{},
	}

	// Check each resource type
	if err := cv.checkPolicies(report); err != nil {
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("Policy check failed: %v", err))
		report.Success = false
	}

	if err := cv.checkRoles(report); err != nil {
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("Role check failed: %v", err))
		report.Success = false
	}

	if err := cv.checkTokens(report); err != nil {
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("Token check failed: %v", err))
		report.Success = false
	}

	if err := cv.checkServiceAccounts(report); err != nil {
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("Service account check failed: %v", err))
		report.Success = false
	}

	// If orphaned resources found, attempt cleanup
	if len(report.OrphanedResources) > 0 {
		cv.performCleanup(report)
	}

	return report, nil
}

// checkPolicies verifies no test policies remain
func (cv *CleanupVerifier) checkPolicies(report *CleanupReport) error {
	policies, err := cv.client.ListPolicies(cv.permissionSystemID)
	if err != nil {
		return fmt.Errorf("failed to list policies: %w", err)
	}

	count := 0
	for _, policy := range policies {
		if cv.isTestResource(policy.Name) {
			createdAt, _ := time.Parse(time.RFC3339, policy.CreatedAt)
			report.OrphanedResources = append(report.OrphanedResources, OrphanedResource{
				Type:       "policy",
				ID:         policy.ID,
				Name:       policy.Name,
				CreatedAt:  createdAt,
				TestPrefix: cv.getTestPrefix(policy.Name),
			})
		}
		count++
	}

	report.ResourceCounts["policies"] = count
	return nil
}

// checkRoles verifies no test roles remain
func (cv *CleanupVerifier) checkRoles(report *CleanupReport) error {
	roles, err := cv.client.ListRoles(cv.permissionSystemID)
	if err != nil {
		return fmt.Errorf("failed to list roles: %w", err)
	}

	count := 0
	for _, role := range roles {
		if cv.isTestResource(role.Name) {
			createdAt, _ := time.Parse(time.RFC3339, role.CreatedAt)
			report.OrphanedResources = append(report.OrphanedResources, OrphanedResource{
				Type:       "role",
				ID:         role.ID,
				Name:       role.Name,
				CreatedAt:  createdAt,
				TestPrefix: cv.getTestPrefix(role.Name),
			})
		}
		count++
	}

	report.ResourceCounts["roles"] = count
	return nil
}

// checkServiceAccounts verifies no test service accounts remain
func (cv *CleanupVerifier) checkServiceAccounts(report *CleanupReport) error {
	serviceAccounts, err := cv.client.ListServiceAccounts(cv.permissionSystemID)
	if err != nil {
		return fmt.Errorf("failed to list service accounts: %w", err)
	}

	count := 0
	for _, sa := range serviceAccounts {
		if cv.isTestResource(sa.Name) {
			createdAt, _ := time.Parse(time.RFC3339, sa.CreatedAt)
			report.OrphanedResources = append(report.OrphanedResources, OrphanedResource{
				Type:       "service_account",
				ID:         sa.ID,
				Name:       sa.Name,
				CreatedAt:  createdAt,
				TestPrefix: cv.getTestPrefix(sa.Name),
			})
		}
		count++
	}

	report.ResourceCounts["service_accounts"] = count
	return nil
}

// checkTokens verifies no test tokens remain
func (cv *CleanupVerifier) checkTokens(report *CleanupReport) error {
	// Get all service accounts first to check their tokens
	serviceAccounts, err := cv.client.ListServiceAccounts(cv.permissionSystemID)
	if err != nil {
		return fmt.Errorf("failed to list service accounts for token check: %w", err)
	}

	totalTokens := 0
	for _, sa := range serviceAccounts {
		tokens, err := cv.client.ListTokens(cv.permissionSystemID, sa.ID)
		if err != nil {
			log.Printf("Warning: failed to list tokens for service account %s: %v", sa.ID, err)
			continue
		}

		for _, token := range tokens {
			if cv.isTestResource(token.Name) {
				createdAt, _ := time.Parse(time.RFC3339, token.CreatedAt)
				report.OrphanedResources = append(report.OrphanedResources, OrphanedResource{
					Type:       "token",
					ID:         token.ID,
					Name:       token.Name,
					CreatedAt:  createdAt,
					TestPrefix: cv.getTestPrefix(token.Name),
				})
			}
			totalTokens++
		}
	}

	report.ResourceCounts["tokens"] = totalTokens
	return nil
}

// isTestResource checks if a resource name matches test prefixes
func (cv *CleanupVerifier) isTestResource(name string) bool {
	for _, prefix := range cv.testPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// getTestPrefix returns the matching test prefix for a resource name
func (cv *CleanupVerifier) getTestPrefix(name string) string {
	for _, prefix := range cv.testPrefixes {
		if strings.HasPrefix(name, prefix) {
			return prefix
		}
	}
	return ""
}

// performCleanup attempts to clean up orphaned resources
func (cv *CleanupVerifier) performCleanup(report *CleanupReport) {
	// Clean up in proper order: tokens -> policies -> roles -> service accounts
	cv.cleanupTokens(report)
	cv.cleanupPolicies(report)
	cv.cleanupRoles(report)
	cv.cleanupServiceAccounts(report)
}

// cleanupTokens removes orphaned tokens
func (cv *CleanupVerifier) cleanupTokens(report *CleanupReport) {
	for _, resource := range report.OrphanedResources {
		if resource.Type != "token" {
			continue
		}

		action := CleanupAction{
			Action:       "delete",
			ResourceType: "token",
			ResourceID:   resource.ID,
			Timestamp:    time.Now(),
		}

		// Find the service account for this token
		serviceAccounts, err := cv.client.ListServiceAccounts(cv.permissionSystemID)
		if err != nil {
			action.Success = false
			action.Error = fmt.Sprintf("failed to list service accounts: %v", err)
			report.CleanupActions = append(report.CleanupActions, action)
			continue
		}

		var serviceAccountID string
		for _, sa := range serviceAccounts {
			tokens, err := cv.client.ListTokens(cv.permissionSystemID, sa.ID)
			if err != nil {
				continue
			}
			for _, token := range tokens {
				if token.ID == resource.ID {
					serviceAccountID = sa.ID
					break
				}
			}
			if serviceAccountID != "" {
				break
			}
		}

		if serviceAccountID == "" {
			action.Success = false
			action.Error = "service account not found for token"
			report.CleanupActions = append(report.CleanupActions, action)
			continue
		}

		err = cv.client.DeleteToken(cv.permissionSystemID, serviceAccountID, resource.ID)
		if err != nil {
			action.Success = false
			action.Error = err.Error()
		} else {
			action.Success = true
		}

		report.CleanupActions = append(report.CleanupActions, action)
	}
}

// cleanupPolicies removes orphaned policies
func (cv *CleanupVerifier) cleanupPolicies(report *CleanupReport) {
	for _, resource := range report.OrphanedResources {
		if resource.Type != "policy" {
			continue
		}

		action := CleanupAction{
			Action:       "delete",
			ResourceType: "policy",
			ResourceID:   resource.ID,
			Timestamp:    time.Now(),
		}

		err := cv.client.DeletePolicy(cv.permissionSystemID, resource.ID)
		if err != nil {
			action.Success = false
			action.Error = err.Error()
		} else {
			action.Success = true
		}

		report.CleanupActions = append(report.CleanupActions, action)
	}
}

// cleanupRoles removes orphaned roles
func (cv *CleanupVerifier) cleanupRoles(report *CleanupReport) {
	for _, resource := range report.OrphanedResources {
		if resource.Type != "role" {
			continue
		}

		action := CleanupAction{
			Action:       "delete",
			ResourceType: "role",
			ResourceID:   resource.ID,
			Timestamp:    time.Now(),
		}

		err := cv.client.DeleteRole(cv.permissionSystemID, resource.ID)
		if err != nil {
			action.Success = false
			action.Error = err.Error()
		} else {
			action.Success = true
		}

		report.CleanupActions = append(report.CleanupActions, action)
	}
}

// cleanupServiceAccounts removes orphaned service accounts
func (cv *CleanupVerifier) cleanupServiceAccounts(report *CleanupReport) {
	for _, resource := range report.OrphanedResources {
		if resource.Type != "service_account" {
			continue
		}

		action := CleanupAction{
			Action:       "delete",
			ResourceType: "service_account",
			ResourceID:   resource.ID,
			Timestamp:    time.Now(),
		}

		err := cv.client.DeleteServiceAccount(cv.permissionSystemID, resource.ID)
		if err != nil {
			action.Success = false
			action.Error = err.Error()
		} else {
			action.Success = true
		}

		report.CleanupActions = append(report.CleanupActions, action)
	}
}

// GetCommonTestPrefixes returns common test prefixes used across test suites
func GetCommonTestPrefixes() []string {
	return []string{
		"test-policy-",
		"test-role-",
		"test-sa-",
		"test-token-",
		"policy-ds-",
		"role-ds-",
		"sa-ds-",
		"token-ds-",
		"policy-list-",
		"role-list-",
		"sa-list-",
		"token-list-",
		"test-",
	}
}

// PerformPostTestCleanup is a convenience function for post-test cleanup
func PerformPostTestCleanup() error {
	permissionSystemID := GetTestPermissionSystemID()
	if permissionSystemID == "" {
		return fmt.Errorf("permission system ID not configured")
	}

	verifier, err := NewCleanupVerifier(permissionSystemID, GetCommonTestPrefixes())
	if err != nil {
		return fmt.Errorf("failed to create cleanup verifier: %w", err)
	}

	report, err := verifier.VerifyCleanup()
	if err != nil {
		return fmt.Errorf("cleanup verification failed: %w", err)
	}

	if !report.Success {
		return fmt.Errorf("cleanup verification failed with %d errors: %v",
			len(report.ErrorMessages), report.ErrorMessages)
	}

	if len(report.OrphanedResources) > 0 {
		log.Printf("Cleaned up %d orphaned resources", len(report.OrphanedResources))
	}

	return nil
}

// LogCleanupReport logs a detailed cleanup report
func LogCleanupReport(report *CleanupReport) {
	log.Printf("=== Cleanup Report ===")
	log.Printf("Permission System: %s", report.PermissionSystemID)
	log.Printf("Timestamp: %s", report.Timestamp.Format(time.RFC3339))
	log.Printf("Success: %t", report.Success)

	log.Printf("Resource Counts:")
	for resourceType, count := range report.ResourceCounts {
		log.Printf("  %s: %d", resourceType, count)
	}

	if len(report.OrphanedResources) > 0 {
		log.Printf("Orphaned Resources (%d):", len(report.OrphanedResources))
		for _, resource := range report.OrphanedResources {
			log.Printf("  %s: %s (%s) - %s", resource.Type, resource.Name, resource.ID, resource.TestPrefix)
		}
	}

	if len(report.CleanupActions) > 0 {
		log.Printf("Cleanup Actions (%d):", len(report.CleanupActions))
		for _, action := range report.CleanupActions {
			status := "✓"
			if !action.Success {
				status = "✗"
			}
			log.Printf("  %s %s %s (%s)", status, action.Action, action.ResourceType, action.ResourceID)
			if action.Error != "" {
				log.Printf("    Error: %s", action.Error)
			}
		}
	}

	if len(report.ErrorMessages) > 0 {
		log.Printf("Errors (%d):", len(report.ErrorMessages))
		for _, errMsg := range report.ErrorMessages {
			log.Printf("  %s", errMsg)
		}
	}

	log.Printf("=== End Cleanup Report ===")
}
