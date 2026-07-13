---
page_title: "Resource: authzed_materialize_deployment"
description: |-
  Manages AuthZed Materialize deployments for a permission system deployment.
---

# authzed_materialize_deployment

This resource allows you to create, update, and delete AuthZed Materialize
deployments. A Materialize deployment continuously materializes the
permissions you specify from a permission system deployment, serving them
from a low-latency cache.

~> **Warning:** This resource uses the internal API version. The provider
sends the required `X-API-Version: internal` header for Materialize
operations automatically, regardless of the `api_version` configured on the
provider. The internal API comes with no compatibility guarantees: it can
change or break at any time without notice, and this resource may stop
working until a provider update restores compatibility. Promotion of these
endpoints to the public API is planned to align with the Materialize GA
release.

## Example Usage

```terraform
resource "authzed_materialize_deployment" "example" {
  name                  = "my-materialize"
  permission_system_id  = "ps-123456789"
  deployment_id         = "dp-123456789"
  server_template_id    = "mtsc-4-vcpu"
  snapshot_template_id  = "mtssc-4-vcpu"
  hydration_template_id = "mthc-4-vcpu"

  watched_permissions = [
    "document#view@user",
  ]

  replicas            = 2
  accelerated_queries = true

  timeouts {
    create = "45m"
    update = "45m"
  }
}
```

## Argument Reference

- `name` - (Required) Name of the Materialize deployment. Must follow the
  RFC-1123 naming convention (lowercase alphanumerics and hyphens, max 50
  characters). Changing it forces a new deployment.
- `permission_system_id` - (Required) ID of the permission system to
  materialize permissions from (`ps-...`). Changing it forces a new
  deployment.
- `deployment_id` - (Required) ID of the permission system deployment that
  provides incremental updates (`dp-...`). Changing it forces a new
  deployment.
- `server_template_id` - (Required) ID of the Materialize online hydration
  server template (e.g. `mtsc-4-vcpu` — see
  [Available Templates](#available-templates)).
- `snapshot_template_id` - (Required) ID of the Materialize snapshot
  template (e.g. `mtssc-4-vcpu` — see
  [Available Templates](#available-templates)).
- `hydration_template_id` - (Required) ID of the Materialize offline
  hydration template (e.g. `mthc-4-vcpu` — see
  [Available Templates](#available-templates)).
- `watched_permissions` - (Required) List of permissions to materialize, in
  the format `resource_type#relation@subject_type[#subject_relation]`
  (1-100 entries). Updates replace the entire list.
- `replicas` - (Optional) Number of online hydration server replicas
  (0-16). Zero temporarily disables the deployment without deleting it.
  When omitted, the server template default applies and the applied value
  is read back into state.
- `accelerated_queries` - (Optional) When `true`, this deployment
  accelerates the permission system's `CheckPermission`, `LookupResources`,
  and `LookupSubjects` queries by serving as a hedged, lower-latency cache.
  When omitted, the value is read back from the deployment status.
- `timeouts` - (Optional) Operation timeouts. Creating and updating a
  Materialize deployment waits until the deployment is provisioned and its
  snapshot job has started — not until it is fully hydrated and serving,
  which is data-dependent and can take hours. Both default to 30 minutes.
  Deletion is asynchronous and bounded by the provider-level
  `delete_timeout`.

## Available Templates

Template IDs follow the pattern `<prefix>-<N>-vcpu`, where `N` is the number
of vCPUs provisioned per replica of that component. The following sizes are
available:

| Size (vCPU) | Server (`server_template_id`) | Snapshot (`snapshot_template_id`) | Hydration (`hydration_template_id`) |
|---|---|---|---|
| 1  | `mtsc-1-vcpu`  | `mtssc-1-vcpu`  | `mthc-1-vcpu`  |
| 2  | `mtsc-2-vcpu`  | `mtssc-2-vcpu`  | `mthc-2-vcpu`  |
| 4  | `mtsc-4-vcpu`  | `mtssc-4-vcpu`  | `mthc-4-vcpu`  |
| 8  | `mtsc-8-vcpu`  | `mtssc-8-vcpu`  | `mthc-8-vcpu`  |
| 16 | `mtsc-16-vcpu` | `mtssc-16-vcpu` | `mthc-16-vcpu` |
| 32 | `mtsc-32-vcpu` | `mtssc-32-vcpu` | `mthc-32-vcpu` |

~> **Note:** The 1 vCPU size is not available on Azure-based environments,
which start at 2 vCPU. The available set can vary by environment and may
change over time.

## Attribute Reference

- `id` - Unique identifier of the Materialize deployment (`mc-...`).
- `url` - Endpoint URL of the Materialize deployment.
- `created_at` - Timestamp when the Materialize deployment was created.

## Import

Materialize deployments can be imported using their ID:

```shell
terraform import authzed_materialize_deployment.example mc-123456789
```
