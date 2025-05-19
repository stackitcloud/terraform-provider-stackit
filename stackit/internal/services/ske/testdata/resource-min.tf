variable "project_id" {}
variable "organization_id" {}
variable "name" {}
variable "nodepool_availability_zone1" {}
variable "nodepool_machine_type" {}
variable "nodepool_maximum" {}
variable "nodepool_minimum" {}
variable "nodepool_name" {}
variable "kubernetes_version_min" {}
variable "maintenance_enable_kubernetes_version_updates" {}
variable "maintenance_enable_machine_image_version_updates" {}
variable "maintenance_start" {}
variable "maintenance_end" {}
variable "region" {}
variable "dns_zone_name" {}
variable "dns_name" {}


resource "stackit_ske_cluster" "cluster" {
  project_id = var.project_id
  name       = var.name

  node_pools = [{
    availability_zones = [var.nodepool_availability_zone1]
    machine_type       = var.nodepool_machine_type
    maximum            = var.nodepool_maximum
    minimum            = var.nodepool_minimum
    name               = var.nodepool_name
    # os_name         = var.nodepool_os_name
    # os_version_min  = var.nodepool_os_version_min

    }
  ]
  kubernetes_version_min = var.kubernetes_version_min
  # even though the maintenance attribute is not mandatory,
  # it is required for a consistent plan
  # see https://jira.schwarz/browse/STACKITTPR-242
  maintenance = {
    enable_kubernetes_version_updates    = var.maintenance_enable_kubernetes_version_updates
    enable_machine_image_version_updates = var.maintenance_enable_machine_image_version_updates
    start                                = var.maintenance_start
    end                                  = var.maintenance_end
  }
  region = var.region
}

resource "stackit_ske_kubeconfig" "kubeconfig" {
  project_id   = stackit_ske_cluster.cluster.project_id
  cluster_name = stackit_ske_cluster.cluster.name
}

data "stackit_ske_cluster" "cluster" {
  project_id = var.project_id
  name       = stackit_ske_cluster.cluster.name
}

resource "stackit_dns_zone" "dns-zone" {
  project_id = var.project_id
  name       = var.dns_zone_name
  dns_name   = var.dns_name
}

