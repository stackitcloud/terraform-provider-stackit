---
page_title: "Using Kubernetes Provider with STACKIT SKE"
---
# Using Kubernetes Provider with STACKIT SKE

## Overview

This guide outlines the process of utilizing the [HashiCorp Kubernetes provider](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs) alongside the STACKIT provider to create and manage resources in a STACKIT SKE Cluster.

## Steps

1. **Configure STACKIT Provider**

    First, configure the STACKIT provider to connect to the STACKIT services.

    ```hcl
    provider "stackit" {
      default_region = "eu01"
    }
    ```

2. **Create STACKIT SKE Cluster**

    Define and create the STACKIT SKE cluster resource with the necessary specifications.

    ```hcl
    resource "stackit_ske_cluster" "ske_cluster_01" {
      project_id             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      name                   = "example-cluster"
      kubernetes_version_min = "1.31"
      
      node_pools = [
        {
          name               = "example-node-pool"
          machine_type       = "g1.3"
          minimum            = 1
          maximum            = 2
          availability_zones = ["eu01-1"]
          os_version_min     = "3815.2.5"
          os_name            = "flatcar"
          volume_size        = 32
          volume_type        = "storage_premium_perf6"
        }
      ]
    }
    ```

3. **Define STACKIT SKE Kubeconfig**

    Create a resource to obtain the kubeconfig for the newly created STACKIT SKE cluster.

    ```hcl
    resource "stackit_ske_kubeconfig" "ske_kubeconfig_01" {
      project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      cluster_name = stackit_ske_cluster.ske_cluster_01.name
      refresh      = true
    }
    ```

4. **Configure Kubernetes Provider**

    Use the kubeconfig from the STACKIT SKE cluster to configure the Kubernetes provider.

    ```hcl
    provider "kubernetes" {
      host                   = yamldecode(stackit_ske_kubeconfig.ske_kubeconfig_01.kube_config).clusters[0].cluster.server
      client_certificate     = base64decode(yamldecode(stackit_ske_kubeconfig.ske_kubeconfig_01.kube_config).users[0].user["client-certificate-data"])
      client_key             = base64decode(yamldecode(stackit_ske_kubeconfig.ske_kubeconfig_01.kube_config).users[0].user["client-key-data"])
      cluster_ca_certificate = base64decode(yamldecode(stackit_ske_kubeconfig.ske_kubeconfig_01.kube_config).clusters[0].cluster["certificate-authority-data"])
    }
    ```

5. **Define Kubernetes Resources**

    Now you can start defining Kubernetes resources that you want to manage. Here is an example of creating a Kubernetes Namespace.

    ```hcl
    resource "kubernetes_namespace" "dev" {
      metadata {
        name = "dev"
      }
    }
    ```