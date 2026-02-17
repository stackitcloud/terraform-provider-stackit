variable "project_id" {}
variable "name" {}
variable "nodepool_availability_zone1" {}
variable "nodepool_machine_type" {}
variable "nodepool_maximum" {}
variable "nodepool_minimum" {}
variable "nodepool_name" {}
variable "nodepool_allow_system_components" {}
variable "nodepool_cri" {}
variable "nodepool_label_value" {}
variable "nodepool_max_surge" {}
variable "nodepool_max_unavailable" {}
variable "nodepool_os_name" {}
variable "nodepool_os_version_min" {}
variable "nodepool_taints_effect" {}
variable "nodepool_taints_key" {}
variable "nodepool_taints_value" {}
variable "nodepool_volume_size" {}
variable "nodepool_volume_type" {}
variable "ext_acl_enabled" {}
variable "ext_acl_allowed_cidr1" {}
variable "ext_observability_enabled" {}
variable "ext_dns_enabled" {}
variable "nodepool_hibernations1_start" {}
variable "nodepool_hibernations1_end" {}
variable "nodepool_hibernations1_timezone" {}
variable "kubernetes_version_min" {}
variable "maintenance_enable_kubernetes_version_updates" {}
variable "maintenance_enable_machine_image_version_updates" {}
variable "maintenance_start" {}
variable "maintenance_end" {}
variable "region" {}
variable "expiration" {}
variable "refresh" {}
variable "refresh_before" {}
variable "dns_zone_name" {}
variable "dns_name" {}
variable "network_control_plane_access_scope" {}

resource "stackit_network" "network" {
  project_id = var.project_id
  name       = var.name
}

resource "stackit_ske_cluster" "cluster" {
  project_id = var.project_id
  name       = var.name

  node_pools = [{
    availability_zones = [var.nodepool_availability_zone1]
    machine_type       = var.nodepool_machine_type
    maximum            = var.nodepool_maximum
    minimum            = var.nodepool_minimum
    name               = var.nodepool_name

    allow_system_components = var.nodepool_allow_system_components
    cri                     = var.nodepool_cri
    labels = {
      "label_key" = var.nodepool_label_value
    }
    max_surge       = var.nodepool_max_surge
    max_unavailable = var.nodepool_max_unavailable
    os_name         = var.nodepool_os_name
    os_version_min  = var.nodepool_os_version_min
    taints = [{
      effect = var.nodepool_taints_effect
      key    = var.nodepool_taints_key
      value  = var.nodepool_taints_value
    }]
    volume_size = var.nodepool_volume_size
    volume_type = var.nodepool_volume_type
    }
  ]

  extensions = {
    acl = {
      enabled       = var.ext_acl_enabled
      allowed_cidrs = [var.ext_acl_allowed_cidr1]
    }
    observability = {
      enabled = var.ext_observability_enabled
    }
    dns = {
      enabled = var.ext_dns_enabled
      zones   = [stackit_dns_zone.dns-zone.dns_name]
    }
  }
  hibernations = [{
    start    = var.nodepool_hibernations1_start
    end      = var.nodepool_hibernations1_end
    timezone = var.nodepool_hibernations1_timezone
  }]
  kubernetes_version_min = var.kubernetes_version_min
  maintenance = {
    enable_kubernetes_version_updates    = var.maintenance_enable_kubernetes_version_updates
    enable_machine_image_version_updates = var.maintenance_enable_machine_image_version_updates
    start                                = var.maintenance_start
    end                                  = var.maintenance_end
  }
  region = var.region
  network = {
    id = stackit_network.network.network_id
    control_plane = {
      access_scope = var.network_control_plane_access_scope
    }
  }
}

resource "stackit_ske_kubeconfig" "kubeconfig" {
  project_id     = stackit_ske_cluster.cluster.project_id
  cluster_name   = stackit_ske_cluster.cluster.name
  expiration     = var.expiration
  refresh        = var.refresh
  refresh_before = var.refresh_before
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

