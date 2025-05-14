package models

// ServiceAccount represents an access management service account for a permission system
type ServiceAccount struct {
	ID                  string  `json:"id,omitempty"`
	Name                string  `json:"name"`
	Description         string  `json:"description,omitempty"`
	PermissionsSystemID string  `json:"permissionsSystemID"`
	Tokens              []Token `json:"token,omitempty"`
	CreatedAt           string  `json:"createdAt,omitempty"`
	Creator             string  `json:"creator,omitempty"`
	ConfigETag          string  `json:"configETag,omitempty"`
}

// Token represents a token associated with a service account
type Token struct {
	ID                  string `json:"id,omitempty"`
	Name                string `json:"name"`
	Description         string `json:"description,omitempty"`
	PermissionsSystemID string `json:"permissionsSystemID"`
	ServiceAccountID    string `json:"serviceAccountID"`
	Hash                string `json:"hash,omitempty"`
	CreatedAt           string `json:"createdAt,omitempty"`
	Creator             string `json:"creator,omitempty"`
}
