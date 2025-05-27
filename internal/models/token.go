package models

// TokenRequest represents a service account token
type TokenRequest struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	PermissionsSystemID string `json:"permissionsSystemID"`
	ServiceAccountID    string `json:"serviceAccountID"`
	CreatedAt           string `json:"createdAt"`
	Creator             string `json:"creator"`
	UpdatedAt           string `json:"updatedAt"`
	Updater             string `json:"updater"`
	Hash                string `json:"hash"`              // The SHA256 hash of the token
	Secret              string `json:"secret"`            // The token value from API (used for plain_text)
	ReturnPlainText     bool   `json:"return_plain_text"` // Request plain text token during creation
}
