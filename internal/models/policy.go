package models

type Policy struct {
	ID                 string   `json:"id,omitempty"`
	Name               string   `json:"name"`
	Description        string   `json:"description,omitempty"`
	PermissionSystemID string   `json:"permissionSystemID"`
	PrincipalID        string   `json:"principalID"`
	RoleIDs            []string `json:"roleIDs"`
	CreatedAt          string   `json:"createdAt,omitempty"`
	Creator            string   `json:"creator,omitempty"`
}
