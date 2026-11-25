variable "project_id" {}
variable "name" {}
variable "description" {}
variable "docker_username" {}
variable "docker_password" {}
variable "docker_password_version" {}

resource "stackit_containerregistry_artifactory" "this" {
  project_id = var.project_id
  name       = var.name
  proxy = {
    registry_type = "docker-hub"
    url           = "https://registry-1.docker.io"
    description   = var.description
    credentials = {
      access_key               = var.docker_username
      access_secret_wo         = var.docker_password
      access_secret_wo_version = var.docker_password_version
    }
  }
}
