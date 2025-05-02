package client

import "terraform-provider-authzed/internal/models"

// NewServiceAccountResource creates a ServiceAccountWithETag Resource
func NewServiceAccountResource(decoded any, etag string) Resource {
	serviceAccount, ok := decoded.(*models.ServiceAccount)
	if !ok {
		panic("Invalid type for ServiceAccount")
	}
	return &ServiceAccountWithETag{
		ServiceAccount: serviceAccount,
		ETag:           etag,
	}
}

// NewRoleResource creates a RoleWithETag Resource
func NewRoleResource(decoded any, etag string) Resource {
	role, ok := decoded.(*models.Role)
	if !ok {
		panic("Invalid type for Role")
	}
	return &RoleWithETag{
		Role: role,
		ETag: etag,
	}
}

// NewPolicyResource creates a PolicyWithETag Resource
func NewPolicyResource(decoded any, etag string) Resource {
	policy, ok := decoded.(*models.Policy)
	if !ok {
		panic("Invalid type for Policy")
	}
	return &PolicyWithETag{
		Policy: policy,
		ETag:   etag,
	}
}

// NewTokenResource creates a TokenWithETag Resource
func NewTokenResource(decoded any, etag string) Resource {
	token, ok := decoded.(*models.Token)
	if !ok {
		panic("Invalid type for Token")
	}
	return &TokenWithETag{
		Token: token,
		ETag:  etag,
	}
}

// NewPermissionsSystemResource creates a PermissionsSystemWithETag Resource
func NewPermissionsSystemResource(decoded any, etag string) Resource {
	permissionsSystem, ok := decoded.(*models.PermissionsSystem)
	if !ok {
		panic("Invalid type for PermissionsSystem")
	}
	return &PermissionsSystemWithETag{
		PermissionsSystem: permissionsSystem,
		ETag:              etag,
	}
}
