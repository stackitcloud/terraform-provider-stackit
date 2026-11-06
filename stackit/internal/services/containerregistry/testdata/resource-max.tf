variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "docker_username" {
  type = string
}

variable "docker_password" {
  type      = string
  sensitive = true
}

variable "docker_password_version" {
  type    = number
  default = 1
}

resource "stackit_containerregistry_artifactory" "this" {
  project_id = var.project_id
  name       = var.name
  proxy = {
    registry_type = "docker-hub"
    url           = "https://registry-1.docker.io"
    description   = "Docker Hub mirror"
    credentials = {
      access_key               = var.docker_username
      access_secret_wo         = var.docker_password
      access_secret_wo_version = var.docker_password_version
    }
  }
}