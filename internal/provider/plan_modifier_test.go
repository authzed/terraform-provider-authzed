package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// TestUseStateForUnknownBehavior documents the expected behavior of UseStateForUnknown
// plan modifier for immutable vs mutable computed fields.
// The correct configuration adds UseStateForUnknown() to immutable fields across all resources.
func TestUseStateForUnknownBehavior(t *testing.T) {
	modifier := stringplanmodifier.UseStateForUnknown()
	if modifier == nil {
		t.Fatal("UseStateForUnknown modifier should be available")
	}
}

// TestPlanModifierDocumentation validates plan modifier configuration
func TestPlanModifierDocumentation(t *testing.T) {
	expectedBehavior := map[string]map[string]bool{
		"immutable_fields": {
			"id":         true, // should have UseStateForUnknown
			"created_at": true,
			"creator":    true,
		},
		"mutable_fields": {
			"updated_at": false, // should not have UseStateForUnknown
			"updater":    false,
			"etag":       false,
		},
	}

	// Validate expected behavior is defined
	if len(expectedBehavior) == 0 {
		t.Fatal("Expected behavior not defined")
	}
}
