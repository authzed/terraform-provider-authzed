package helpers

import (
	"fmt"
	"os"
	"strings"
)

// GetTestDeploymentID returns the permission system deployment (dp-...) that
// Materialize acceptance tests target. When unset, the Materialize
// acceptance tests skip.
func GetTestDeploymentID() string {
	return os.Getenv("AUTHZED_DEPLOYMENT_ID")
}

// envOrDefault returns the env var's value, or fallback when unset/empty.
func envOrDefault(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

// The template fallbacks below (mtsc-default/mtssc-default/mthc-default)
// only exist in AuthZed's local development stacks, so they are useful to
// maintainers testing locally without extra configuration. Runs against any
// real environment must set the AUTHZED_MATERIALIZE_*_TEMPLATE_ID variables
// to templates that exist there (e.g. mtsc-1-vcpu).

// GetTestMaterializeServerTemplateID returns the online hydration server template ID
func GetTestMaterializeServerTemplateID() string {
	return envOrDefault("AUTHZED_MATERIALIZE_SERVER_TEMPLATE_ID", "mtsc-default")
}

// GetTestMaterializeSnapshotTemplateID returns the snapshot template ID
func GetTestMaterializeSnapshotTemplateID() string {
	return envOrDefault("AUTHZED_MATERIALIZE_SNAPSHOT_TEMPLATE_ID", "mtssc-default")
}

// GetTestMaterializeHydrationTemplateID returns the offline hydration template ID
func GetTestMaterializeHydrationTemplateID() string {
	return envOrDefault("AUTHZED_MATERIALIZE_HYDRATION_TEMPLATE_ID", "mthc-default")
}

// GetTestMaterializeWatchedPermissions returns the comma-separated watched
// permissions (resource_type#relation@subject_type) valid for the test
// permission system's schema. Empty means Materialize tests should skip.
func GetTestMaterializeWatchedPermissions() []string {
	raw := os.Getenv("AUTHZED_MATERIALIZE_WATCHED_PERMISSIONS")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	watched := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			watched = append(watched, trimmed)
		}
	}
	return watched
}

// BuildMaterializeDeploymentConfig builds HCL for a materialize deployment
// with the given name and watched permissions
func BuildMaterializeDeploymentConfig(name string, watched []string) string {
	return buildMaterializeDeploymentConfig(name, watched, "")
}

// BuildMaterializeDeploymentConfigWithOptions builds HCL for a materialize
// deployment that also pins the optional replicas and accelerated_queries
// attributes, exercising their round-trip through deployment status.
func BuildMaterializeDeploymentConfigWithOptions(name string, watched []string, replicas int64, acceleratedQueries bool) string {
	extraAttrs := fmt.Sprintf("  replicas              = %d\n  accelerated_queries   = %t\n", replicas, acceleratedQueries)
	return buildMaterializeDeploymentConfig(name, watched, extraAttrs)
}

// buildMaterializeDeploymentConfig renders the shared resource block;
// extraAttrs is appended verbatim inside it (one full attribute line each,
// newline-terminated).
func buildMaterializeDeploymentConfig(name string, watched []string, extraAttrs string) string {
	quoted := make([]string, 0, len(watched))
	for _, w := range watched {
		quoted = append(quoted, fmt.Sprintf("%q", w))
	}
	return fmt.Sprintf(
		`
%s

resource "authzed_materialize_deployment" "test" {
  name                  = %q
  permission_system_id  = %q
  deployment_id         = %q
  server_template_id    = %q
  snapshot_template_id  = %q
  hydration_template_id = %q
  watched_permissions   = [%s]
%s}
`,
		BuildProviderConfig(),
		name,
		GetTestPermissionSystemID(),
		GetTestDeploymentID(),
		GetTestMaterializeServerTemplateID(),
		GetTestMaterializeSnapshotTemplateID(),
		GetTestMaterializeHydrationTemplateID(),
		strings.Join(quoted, ", "),
		extraAttrs,
	)
}
