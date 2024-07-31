---
page_title: "Retrieving SKE outgoing IP address with Terraform"
---
# Retrieving SKE outgoing IP address with Terraform

To retrieve the outgoing IP address of your STACKIT Kubernetes Engine (SKE) cluster, you have two options based on your initial SKE setup.

## Default Setup

If you haven't configured any organisational network settings, you can use the default setup. It is necessary to use the Terraform OpenStack provider in conjunction with the STACKIT provider.

```hcl
resource "stackit_ske_cluster" "ske_cluster_01" {
  project_id             = var.project_id
  name                   = var.cluster_name
  kubernetes_version_min = "1.29.6"
  node_pools = [...]
}

data "openstack_networking_router_v2" "router" {
  name = "shoot--${substr(sha256(var.project_id), 0, 10)}--${var.cluster_name}"
}

locals {
  cluster_outgoing_ip = data.openstack_networking_router_v2.router.external_fixed_ip.0.ip_address
}
```

## Organizational SNA Setup

If your project is within an organizational STACKIT Network Area (SNA), you can attach a `stackit_network` to the SKE cluster:

```hcl
resource "stackit_network" "ske_network" {
  project_id         = var.project_id
  name               = "ske-network"
  nameservers        = ["1.1.1.1", "8.8.8.8"]
  ipv4_prefix_length = 24
}

resource "stackit_ske_cluster" "ske_cluster_01" {
  project_id             = var.project_id
  name                   = var.cluster_name
  kubernetes_version_min = "1.29.6"
  node_pools = [...]

  network = {
    id = stackit_network.ske_network.network_id
  }
}

locals {
  cluster_outgoing_ip = stackit_network.ske_network.public_ip
}
```

In both examples, the attribute `cluster_outgoing_ip` will provide the outgoing IP address of your cluster.
The specific configuration depends on whether your setup is within an organizational SNA or a default setup.