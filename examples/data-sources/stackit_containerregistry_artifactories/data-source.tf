locals {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

data "stackit_containerregistry_artifactories" "this" {
  project_id = local.project_id
}

output "artifactory_ids" {
  value = [for artifactory in data.stackit_containerregistry_artifactories.this.results : artifactory.artifactory_id]
}

# These expressions filter locally in Terraform; they are not API filters.
locals {
  exact_name          = [for artifactory in data.stackit_containerregistry_artifactories.this.results : artifactory if artifactory.name == "my-registry"]
  matching_names      = [for artifactory in data.stackit_containerregistry_artifactories.this.results : artifactory if can(regex("my-.*", artifactory.name))]
  proxy_artifactories = [for artifactory in data.stackit_containerregistry_artifactories.this.results : artifactory if artifactory.proxy != null]
  docker_hub_proxies  = [for artifactory in data.stackit_containerregistry_artifactories.this.results : artifactory if try(artifactory.proxy.registry_type, "") == "docker-hub"]
}
