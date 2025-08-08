# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- Resolved plan modifier inconsistencies introduced with FGAM field additions in v0.4.0
- Fixed "Provider produced inconsistent result after apply" errors by correctly configuring UseStateForUnknown() plan modifiers for immutable vs mutable computed fields
- Immutable fields (id, created_at, creator) now use UseStateForUnknown() to prevent "(known after apply)" noise
- Mutable FGAM fields (updated_at, updater) correctly omit UseStateForUnknown() to display actual API values

## [0.4.1] - 2025-01-08 [YANKED]

### Fixed
- Ensures consistent behavior across all resources and eliminates plan inconsistencies for policy updates

**Note:** This version has been yanked due to incomplete plan modifier configuration. Use v0.5.0+ instead.

## [0.4.0] - 2025-01-08

### Added
- Native FGAM (Fine-Grained Access Management) field support for all resources
- New `updated_at` and `updater` fields for Service Accounts, Roles, Policies, and Tokens
- Enhanced retry logic with support for 429 (Rate Limit) status codes
- Improved concurrent operation handling with increased retry limits

### Fixed
- **CRITICAL**: Resolved "Provider produced inconsistent result after apply" errors caused by etag inconsistencies
- Fixed etag handling by removing incorrect UseStateForUnknown() plan modifiers from etag fields
- Ensured etag values always come from API responses for consistent state management

### Changed
- Removed provider-side FGAM patches and locking mechanisms in favor of native API support
- Updated OpenAPI spec integration to use latest API fixes


### Removed
- Provider-side FGAM coordinators and serialization logic (replaced by native API support)

## [0.1.0] - 2023-08-15

### Added
- Initial release of the Terraform Provider for AuthZed Cloud API
- Support for managing Permission Systems
- Support for managing Roles and Permissions
- Support for managing Service Accounts
- Support for managing Tokens
- Support for managing Policies
- Documentation and examples

[Unreleased]: https://github.com/authzed/terraform-provider-cloudapi/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/authzed/terraform-provider-cloudapi/releases/tag/v0.1.0 