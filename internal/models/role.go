package models

// Role represents a permission role in a permissions system
type Role struct {
	ID                  string            `json:"id,omitempty"`
	PermissionsSystemID string            `json:"permissionsSystemID"`
	Name                string            `json:"name"`
	Description         string            `json:"description,omitempty"`
	Permissions         PermissionExprMap `json:"permissions,omitempty"`
	CreatedAt           string            `json:"createdAt,omitempty"`
	Creator             string            `json:"creator,omitempty"`
	ConfigETag          string            `json:"configETag,omitempty"`
}

type PermissionExprMap map[string]string
