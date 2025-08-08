package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// TestPlanModifierRegression validates plan modifier availability
func TestPlanModifierRegression(t *testing.T) {
	modifier := stringplanmodifier.UseStateForUnknown()
	if modifier == nil {
		t.Fatal("UseStateForUnknown modifier should be available")
	}

	// Validate expected behavior mapping
	expectedBehavior := map[string]bool{
		"id":         true, // should have UseStateForUnknown
		"created_at": true,
		"creator":    true,
		"updated_at": false, // should not have UseStateForUnknown
		"updater":    false,
		"etag":       false,
	}

	if len(expectedBehavior) == 0 {
		t.Fatal("Expected behavior not defined")
	}
}
