variable "parent_container_id" {}
variable "owner_email" {}
variable "name" {}

# no test candidate, just needed for the testing setup
resource "stackit_resourcemanager_project" "project" {
  parent_container_id = var.parent_container_id
  name                = var.name
  labels = {
    "enable-vpc" = "true"
  }
  owner_email = var.owner_email
}

resource "stackit_vpc" "vpc" {
  project_id  = stackit_resourcemanager_project.project.project_id
  name        = var.name
  description = ""
}

# SUT

resource "stackit_vpc_region" "region" {
  project_id = stackit_resourcemanager_project.project.project_id
  vpc_id     = stackit_vpc.vpc.vpc_id
  # region left to default
}