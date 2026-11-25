locals {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

data "stackit_containerregistry_replication_adapters" "this" {
  project_id = local.project_id
}

# These expressions filter locally in Terraform; they are not API filters.
locals {
  docker_hub_adapters = [for adapter in data.stackit_containerregistry_replication_adapters.this.results : adapter if adapter.registry_type == "docker-hub"]
  basic_auth_adapters = [for adapter in data.stackit_containerregistry_replication_adapters.this.results : adapter if adapter.credential_type == "basic"]
}
