package models

// ServiceAccount represents an access management service account for a permission system
type ServiceAccount struct {
	ID                  string         `json:"id,omitempty"`
	Name                string         `json:"name"`
	Description         string         `json:"description,omitempty"`
	PermissionsSystemID string         `json:"permissionsSystemID"`
	Tokens              []TokenRequest `json:"token,omitempty"`
	CreatedAt           string         `json:"createdAt,omitempty"`
	Creator             string         `json:"creator,omitempty"`
	UpdatedAt           string         `json:"updatedAt,omitempty"`
	Updater             string         `json:"updater,omitempty"`
}
