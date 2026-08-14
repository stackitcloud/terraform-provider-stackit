variable "parent_container_id" {}
variable "owner_email" {}
variable "testing_setup_name" {}

variable "ip_version" {}
variable "prefix" {}
variable "description" {}

# no test candidate, just needed for the testing setup
resource "stackit_resourcemanager_project" "project" {
  parent_container_id = var.parent_container_id
  name                = var.testing_setup_name
  labels = {
    "enable-vpc" = "true"
  }
  owner_email = var.owner_email
}

resource "stackit_vpc" "vpc" {
  project_id  = stackit_resourcemanager_project.project.project_id
  name        = var.testing_setup_name
  description = ""
}

resource "stackit_vpc_region" "region" {
  project_id = stackit_resourcemanager_project.project.project_id
  vpc_id     = stackit_vpc.vpc.vpc_id
}

# SUT

resource "stackit_vpc_network_range" "network_range" {
  depends_on  = [stackit_vpc_region.region]
  project_id  = stackit_resourcemanager_project.project.project_id
  vpc_id      = stackit_vpc.vpc.vpc_id
  ip_version  = var.ip_version
  prefix      = var.prefix
  description = var.description
}
