variable "name" {}
variable "organization_id" {}
variable "owner_email" {}
variable "update" {}

# no test candidate, just needed for the testing setup
resource "stackit_resourcemanager_project" "project" {
  parent_container_id = var.organization_id
  name                = "vpc-project"
  labels = {
    "enable-vpc" = "true"
  }
  owner_email = var.owner_email
}

resource "stackit_vpc" "vpc" {
  project_id  = stackit_resourcemanager_project.project.project_id
  name        = "vpc"
  description = ""
}

resource "stackit_vpc" "vpc-update" {
  project_id  = stackit_resourcemanager_project.project.project_id
  name        = "vpc-update"
  description = ""
}

resource "stackit_vpc_region" "region" {
  project_id = stackit_resourcemanager_project.project.project_id
  vpc_id     = stackit_vpc.vpc.vpc_id
}

resource "stackit_vpc_region" "region-update" {
  project_id = stackit_resourcemanager_project.project.project_id
  vpc_id     = stackit_vpc.vpc-update.vpc_id
}

resource "stackit_vpc_network_range" "network_range" {
  depends_on  = [stackit_vpc_region.region]
  project_id  = stackit_resourcemanager_project.project.project_id
  vpc_id      = stackit_vpc.vpc.vpc_id
  ip_version  = "ipv4"
  prefix      = "192.168.1.0/24"
  description = ""
}

resource "stackit_vpc_network_range" "network_range-update" {
  depends_on  = [stackit_vpc_region.region-update]
  project_id  = stackit_resourcemanager_project.project.project_id
  vpc_id      = stackit_vpc.vpc-update.vpc_id
  ip_version  = "ipv4"
  prefix      = "10.0.0.8/24"
  description = ""
}

# SUT
resource "stackit_network" "network_vpc" {
  project_id                = stackit_resourcemanager_project.project.project_id
  name                      = var.name
  ipv4_prefix_length        = 28
  ipv4_vpc_network_range_id = var.update ? stackit_vpc_network_range.network_range-update.network_range_id : stackit_vpc_network_range.network_range.network_range_id
  vpc_id                    = var.update ? stackit_vpc.vpc-update.vpc_id : stackit_vpc.vpc.vpc_id
}
