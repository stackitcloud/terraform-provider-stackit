variable "parent_container_id" {}
variable "owner_email" {}

variable "project_name" {}
variable "name" {}

# no test candidate, just needed for the testing setup
resource "stackit_resourcemanager_project" "project" {
  parent_container_id = var.parent_container_id
  name                = var.project_name
  labels = {
    "enable-vpc" = "true"
  }
  owner_email = var.owner_email
}

resource "stackit_vpc" "vpc" {
  project_id  = stackit_resourcemanager_project.project.project_id
  name        = "my-vpc"
  description = ""
}

resource "stackit_vpc_region" "region" {
  project_id = stackit_resourcemanager_project.project.project_id
  vpc_id     = stackit_vpc.vpc.vpc_id
}

resource "stackit_vpc_routing_table" "routing_table" {
  project_id = stackit_resourcemanager_project.project.project_id
  vpc_id     = stackit_vpc.vpc.vpc_id
  name       = var.name
  depends_on = [stackit_vpc_region.region]
}
