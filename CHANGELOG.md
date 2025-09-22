# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Concurrency testing suite** - Performance benchmarking (15-75 resources), concurrent creation tests, and eventual consistency validation
- **DeleteLanes infrastructure** - Conflict resolution system for resource deletion with intelligent retry logic
- **Per-Permission System serialization lanes (PSLanes)** - Concurrent operations across different permission systems while preventing FGAM conflicts
- **Enhanced troubleshooting documentation** - Performance guidance with resource count thresholds and parallelism recommendations

### Changed
- **Client architecture refactor** - Improved retry mechanisms, exponential backoff, and enhanced context handling
- **Performance optimizations** - Intelligent serialization, wait logic for eventual consistency, and significantly reduced execution time
- **Resource creation flow** - Better context handling to prevent timeout and deadline exceeded errors
- **Performance recommendations** - Default parallelism for ≤8 resources; use `parallelism=1` for >8 mixed resources, >5 service accounts, or >50 total resources
- **HTTP compression disabled by default** - Ensures ETag visibility behind proxies
- **Updated dependencies** - golang.org/x/sync v0.17.0, terraform-plugin-framework v1.16.0, terraform-plugin-framework-timeouts v0.6.0

### Fixed
- **FGAM field drift** - Resolved `updated_at`/`updater` drift with proper UseStateForUnknown plan modifiers
- **Context deadline errors** - Fixed timeout issues in policy/role creation
- **Resource deletion conflicts** - Enhanced conflict handling with DeleteLanes
- **Service account state consistency** - Resolved disappearing resources due to eventual consistency
- **FGAM conflicts** - PSLanes prevent 409 errors within same Permission System
- **Plan modifier inconsistencies** - Fixed "Provider produced inconsistent result after apply" errors
- **ETag header extraction** - Fixed issues with compressed responses
- **Test suite stability** - Removed merge conflict markers breaking CI

### Removed
- **Obsolete FGAM serialization configuration** - Removed deprecated `fgam_serialization` provider option and related documentation (replaced by more sophisticated PSLanes system)
- **Legacy API implementation** - Cleaned up unused internal/api code after client optimizations and refactoring

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