package models

type Role struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Description        string            `json:"description,omitempty"`
	PermissionSystemID string            `json:"permissionSystemID"`
	Permissions        PermissionExprMap `json:"permissions"`
	CreatedAt          string            `json:"createdAt,omitempty"`
	Creator            string            `json:"creator,omitempty"`
}

type PermissionExprMap map[string]string
