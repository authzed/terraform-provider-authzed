package models

// Policy represents a permission policy in a permissions system
type Policy struct {
	ID                  string   `json:"id,omitempty"`
	Name                string   `json:"name"`
	Description         string   `json:"description,omitempty"`
	PermissionsSystemID string   `json:"permissionsSystemID"`
	PrincipalID         string   `json:"principalID"`
	RoleIDs             []string `json:"roleIDs"`
	CreatedAt           string   `json:"createdAt,omitempty"`
	Creator             string   `json:"creator,omitempty"`
	ConfigETag          string   `json:"ConfigETag,omitempty"`
}
