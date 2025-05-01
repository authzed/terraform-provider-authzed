package provider

import (
	"reflect"
	"testing"
)

// TestETagIsDefinedInSchema verifies that the ETag field exists in the resource schema
// This ensures the resource properly supports optimistic concurrency control
func TestETagIsDefinedInSchema(t *testing.T) {
	// Create a service account resource
	resource := NewServiceAccountResource()

	// Verify resource creation succeeded
	if resource == nil {
		t.Fatal("Failed to create service account resource")
	}

	// Verify the resource model has an etag field
	resourceModel := serviceAccountResourceModel{}

	// Use reflection to check the field exists and is tagged correctly
	modelType := reflect.TypeOf(resourceModel)
	field, found := modelType.FieldByName("ETag")

	if !found {
		t.Fatal("ETag field not found in serviceAccountResourceModel")
	}

	// Check the field has the correct tag for Terraform SDK
	if field.Tag.Get("tfsdk") != "etag" {
		t.Errorf("ETag field has incorrect tfsdk tag: %s", field.Tag.Get("tfsdk"))
	}
}
