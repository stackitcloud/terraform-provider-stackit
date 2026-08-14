variable "parent_container_id" {}
variable "owner_email" {}

variable "project_name" {}
variable "name" {}
variable "description" {}
variable "region" {}
variable "dynamic_routes" {}
variable "system_routes" {}
variable "label_key" {}
variable "label_value" {}

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
  project_id     = stackit_resourcemanager_project.project.project_id
  vpc_id         = stackit_vpc.vpc.vpc_id
  name           = var.name
  description    = var.description
  region         = var.region
  dynamic_routes = var.dynamic_routes
  system_routes  = var.system_routes
  labels = {
    (var.label_key) = var.label_value
  }
  depends_on = [stackit_vpc_region.region]
}
