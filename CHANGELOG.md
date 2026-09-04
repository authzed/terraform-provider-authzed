# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.7.0-beta2] - 2026-09-04

### Fixed
- **Rejected Materialize configurations now fail fast** - A deployment whose preflight check rejects its configuration (`InvalidWatchedPermission`, `SchemaEmpty`, `NoWatchedPermissions`) reports that only in `status.conditions`, which the provider could not tell apart from a deployment still starting up. Creates now end the wait immediately and name the offending setting instead of polling to the timeout
- **Updates no longer report success before the new configuration is checked** - An update with a rejected configuration previously succeeded almost immediately and wrote the rejected configuration to state. A change is now trusted only once the status names a newer configuration version and the configuration check has run and finished, with a settle window for changes that need no check
- **Client errors carry the API's field-level detail** - The `errors` array in an API error body was ignored and only the generic summary surfaced, so a malformed watched permission reported `bad request` while the API had explained precisely what was expected. Affects every resource in the provider, not just Materialize. A detail that only repeats the summary is dropped, so errors that were already useful are unchanged

### Changed
- **Timeout messages name the underlying failure** - Any failure the deployment reports is appended to the not-ready message, so a wait that does time out says what went wrong rather than only blaming the timeout
- **Ignore local Terraform working files** - Added `.gitignore` entries for Terraform working directories and env files

## [0.7.0-beta1] - 2026-09-03

### Added
- **New resource `authzed_materialize_deployment`** - Manage AuthZed Materialize deployments as code: create, read, update, delete, and import (`terraform import authzed_materialize_deployment.example mc-...`)
- **Materialize deployment configuration** - Watched permissions, server/snapshot/hydration templates, online hydration replica count (`0` pauses a deployment without deleting it), and accelerated queries
- **In-place Materialize updates** - Templates, watched permissions, replicas, and accelerated queries update in place; changing `name`, `permission_system_id`, or `deployment_id` forces replacement (not supported by the API)
- **Materialize drift detection** - Replicas and accelerated queries are read back from deployment status, and deployments deleted outside Terraform are detected and planned for recreation
- **Configurable Materialize timeouts** - `create`/`update` return once the deployment is provisioned and its snapshot job has started (not once fully hydrated, which is data-dependent and can take hours); both default to 30 minutes. Deletion is asynchronous and polled until the deployment is gone
- **Materialize documentation and examples** - Resource docs with the available template ID matrix, an example configuration, and environment-gated acceptance tests
- **golangci-lint configuration** - Lint rules (`.golangci.yaml`) enforced across the provider and client packages

### Changed
- **OpenAPI spec automation** - The spec is now fetched from the AuthZed `cloud-api` repository as the source of truth, and refreshed to the latest version
- **Updated dependencies** - terraform-plugin-framework v1.19.0, terraform-plugin-framework-timeouts v0.7.0, terraform-plugin-log v0.11.0, terraform-plugin-testing v1.16.0, golang.org/x/sync v0.22.0, testify v1.12.1
- **Updated GitHub Actions** - actions/checkout v7, actions/setup-go v7, goreleaser/goreleaser-action v7, peter-evans/create-pull-request v8

**Warning:** `authzed_materialize_deployment` uses the **internal** API version. The provider sends the required `X-API-Version: internal` header for Materialize operations automatically, regardless of the `api_version` configured on the provider. The internal API carries no compatibility guarantees: it can change or break at any time without notice, and this resource may stop working until a provider update restores compatibility. Promotion of these endpoints to the public API is planned to align with the Materialize GA release.

## [0.6.0-beta1] - 2025-09-22

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

[0.7.0-beta2]: https://github.com/authzed/terraform-provider-authzed/releases/tag/v0.7.0-beta2
[0.7.0-beta1]: https://github.com/authzed/terraform-provider-authzed/releases/tag/v0.7.0-beta1
[0.6.0-beta1]: https://github.com/authzed/terraform-provider-authzed/releases/tag/v0.6.0-beta1
[0.4.1]: https://github.com/authzed/terraform-provider-authzed/releases/tag/v0.4.1
[0.4.0]: https://github.com/authzed/terraform-provider-authzed/releases/tag/v0.4.0
[0.1.0]: https://github.com/authzed/terraform-provider-authzed/releases/tag/v0.1.0
