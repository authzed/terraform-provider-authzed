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
