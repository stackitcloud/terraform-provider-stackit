locals {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  region     = "eu01"
  name       = "my-registry"
}

# Standard container registry artifactory
resource "stackit_containerregistry_artifactory" "this" {
  project_id = local.project_id
  name       = local.name
}

# Container registry artifactory configured as a Docker Hub proxy-cache with write-only credentials
resource "stackit_containerregistry_artifactory" "docker_hub_proxy" {
  project_id = local.project_id
  name       = "my-docker-hub-mirror"

  proxy = {
    registry_type = "docker-hub"
    url           = "https://registry-1.docker.io"
    description   = "Docker Hub pull-through cache"

    credentials = {
      access_key               = "my-dockerhub-username"
      access_secret_wo         = "my-dockerhub-access-token"
      access_secret_wo_version = 1
    }
  }
}
