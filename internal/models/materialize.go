package models

// MaterializePhaseRunning is the deployment phase indicating the Materialize
// deployment is provisioned and serving.
const MaterializePhaseRunning = "Running"

// Materialize job phases reported for the snapshot and hydration jobs.
const (
	MaterializeJobPhaseRunning  = "Running"
	MaterializeJobPhaseComplete = "Complete"
	MaterializeJobPhaseDisabled = "Disabled"
)

// MaterializeDeployment represents a Materialize deployment as returned by the API
type MaterializeDeployment struct {
	ID                  string                       `json:"id"`
	Name                string                       `json:"name"`
	PermissionsSystemID string                       `json:"permissionsSystemID"`
	DeploymentID        string                       `json:"deploymentID"`
	ServerTemplateID    string                       `json:"serverTemplateID,omitempty"`
	SnapshotTemplateID  string                       `json:"snapshotTemplateID,omitempty"`
	HydrationTemplateID string                       `json:"hydrationTemplateID,omitempty"`
	WatchedPermissions  []string                     `json:"watchedPermissions"`
	URL                 string                       `json:"url,omitempty"`
	CreatedAt           string                       `json:"createdAt,omitempty"`
	Status              *MaterializeDeploymentStatus `json:"status,omitempty"`
}

// MaterializeDeploymentStatus is the subset of the deployment status the
// provider reads: the health phase, the snapshot job state used as the
// create-readiness signal, plus the feature flags and compute that mirror
// configurable attributes back for drift detection.
type MaterializeDeploymentStatus struct {
	Phase    string                             `json:"phase,omitempty"`
	Snapshot *MaterializeDeploymentSnapshotInfo `json:"snapshot,omitempty"`
	Features *MaterializeDeploymentFeatures     `json:"features,omitempty"`
	Compute  *MaterializeDeploymentCompute      `json:"compute,omitempty"`
}

// MaterializeDeploymentSnapshotInfo reports the state of the snapshot job
type MaterializeDeploymentSnapshotInfo struct {
	Phase string `json:"phase,omitempty"`
}

// MaterializeDeploymentFeatures reports which optional features are enabled
type MaterializeDeploymentFeatures struct {
	AcceleratedQueries bool `json:"acceleratedQueries"`
}

// MaterializeDeploymentCompute is the per-component compute breakdown
type MaterializeDeploymentCompute struct {
	CacheServer *MaterializeComponentCompute `json:"cacheServer,omitempty"`
}

// MaterializeComponentCompute describes one component's provisioned compute
type MaterializeComponentCompute struct {
	Replicas *int64 `json:"replicas,omitempty"`
}

// GetMaterializeDeploymentResponse wraps the deployment returned by GET
type GetMaterializeDeploymentResponse struct {
	Deployment MaterializeDeployment `json:"deployment"`
}

// CreateMaterializeDeploymentRequest is the body for POST /materialize.
// Nil optional pointers are omitted, which the API interprets as "use the
// server template default".
type CreateMaterializeDeploymentRequest struct {
	PermissionsSystemID string   `json:"permissionsSystemID"`
	DeploymentID        string   `json:"deploymentID"`
	ServerTemplateID    string   `json:"serverTemplateID"`
	SnapshotTemplateID  string   `json:"snapshotTemplateID"`
	HydrationTemplateID string   `json:"hydrationTemplateID"`
	Name                string   `json:"name"`
	WatchedPermissions  []string `json:"watchedPermissions"`
	Replicas            *int64   `json:"replicas,omitempty"`
	AcceleratedQueries  *bool    `json:"acceleratedQueries,omitempty"`
}

// CreateMaterializeDeploymentResponse is the body returned by POST /materialize
type CreateMaterializeDeploymentResponse struct {
	ID           string `json:"id"`
	DeploymentID string `json:"deploymentID"`
	Name         string `json:"name"`
}

// UpdateMaterializeDeploymentRequest is the body for PATCH /materialize/{id}.
// Nil pointers are omitted, which the API interprets as "leave unchanged".
type UpdateMaterializeDeploymentRequest struct {
	ServerTemplateID    *string   `json:"serverTemplateID,omitempty"`
	SnapshotTemplateID  *string   `json:"snapshotTemplateID,omitempty"`
	HydrationTemplateID *string   `json:"hydrationTemplateID,omitempty"`
	WatchedPermissions  *[]string `json:"watchedPermissions,omitempty"`
	Replicas            *int64    `json:"replicas,omitempty"`
	AcceleratedQueries  *bool     `json:"acceleratedQueries,omitempty"`
}
