---
page_title: "SKE Log Alerts with STACKIT Observability"
---
# SKE Log Alerts with STACKIT Observability

## Overview

This guide walks you through setting up log-based alerting in STACKIT Observability using Promtail to ship Kubernetes logs.

1. **Set Up Providers**

   Begin by configuring the STACKIT and Kubernetes providers to connect to the STACKIT services.

   ```hcl
   provider "stackit" {
     region = "eu01"
   }

   provider "kubernetes" {
     host                   = yamldecode(stackit_ske_kubeconfig.example.kube_config).clusters.0.cluster.server
     client_certificate     = base64decode(yamldecode(stackit_ske_kubeconfig.example.kube_config).users.0.user.client-certificate-data)
     client_key             = base64decode(yamldecode(stackit_ske_kubeconfig.example.kube_config).users.0.user.client-key-data)
     cluster_ca_certificate = base64decode(yamldecode(stackit_ske_kubeconfig.example.kube_config).clusters.0.cluster.certificate-authority-data)
   }

   provider "helm" {
     kubernetes {
       host                   = yamldecode(stackit_ske_kubeconfig.example.kube_config).clusters.0.cluster.server
       client_certificate     = base64decode(yamldecode(stackit_ske_kubeconfig.example.kube_config).users.0.user.client-certificate-data)
       client_key             = base64decode(yamldecode(stackit_ske_kubeconfig.example.kube_config).users.0.user.client-key-data)
       cluster_ca_certificate = base64decode(yamldecode(stackit_ske_kubeconfig.example.kube_config).clusters.0.cluster.certificate-authority-data)
     }
   }
   ```

2. **Create SKE Cluster and Kubeconfig Resource**

   Set up a STACKIT SKE Cluster and generate the associated kubeconfig resource.

   ```hcl
   resource "stackit_ske_cluster" "example" {
     project_id         = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     name               = "example"
     kubernetes_version = "1.31"
     node_pools = [
       {
         name               = "standard"
         machine_type       = "c1.4"
         minimum            = "3"
         maximum            = "9"
         max_surge          = "3"
         availability_zones = ["eu01-1", "eu01-2", "eu01-3"]
         os_version_min     = "4081.2.1"
         os_name            = "flatcar"
         volume_size        = 32
         volume_type        = "storage_premium_perf6"
       }
     ]
     maintenance = {
       enable_kubernetes_version_updates    = true
       enable_machine_image_version_updates = true
       start                                = "01:00:00Z"
       end                                  = "02:00:00Z"
     }
   }

   resource "stackit_ske_kubeconfig" "example" {
     project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     cluster_name = stackit_ske_cluster.example.name
     refresh      = true
   }
   ```

3. **Create Observability Instance and Credentials**

   Establish a STACKIT Observability instance and its credentials to handle alerts.

   ```hcl
   locals {
     alert_config = {
       route = {
         receiver        = "EmailStackit",
         repeat_interval = "1m",
         continue        = true
       }
       receivers = [
         {
           name = "EmailStackit",
           email_configs = [
             {
               to = "<email>"
             }
           ]
         }
       ]
     }
   }

   resource "stackit_observability_instance" "example" {
     project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     name       = "example"
     plan_name  = "Observability-Large-EU01"
     alert_config = local.alert_config
   }

   resource "stackit_observability_credential" "example" {
     project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     instance_id = stackit_observability_instance.example.instance_id
   }
   ```

4. **Install Promtail**

   Deploy Promtail via Helm to collect logs and forward them to the STACKIT Observability Loki endpoint.

   ```hcl
   resource "helm_release" "promtail" {
     name       = "promtail"
     repository = "https://grafana.github.io/helm-charts"
     chart      = "promtail"
     namespace  = kubernetes_namespace.monitoring.metadata.0.name
     version    = "6.16.4"

     values = [
       <<-EOF
       config:
         clients:
         # To find the Loki push URL, navigate to the observability instance in the portal and select the API tab.
         - url: "https://${stackit_observability_credential.example.username}:${stackit_observability_credential.example.password}@<your-loki-push-url>/instances/${stackit_observability_instance.example.instance_id}/loki/api/v1/push"
       EOF
     ]
   }
   ```

5. **Create Alert Group**

   Create a log alert that triggers when a specific pod logs an error message.

   ```hcl
   resource "stackit_observability_logalertgroup" "example" {
     project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     instance_id = stackit_observability_instance.example.instance_id
     name        = "TestLogAlertGroup"
     interval    = "1m"
     rules = [
       {
         alert      = "SimplePodLogAlertCheck"
         expression = "sum(rate({namespace=\"example\", pod=\"logger\"} |= \"Simulated error message\" [1m])) > 0"
         for        = "60s"
         labels = {
           severity = "critical"
         },
         annotations = {
           summary : "Test Log Alert is working"
           description : "Test Log Alert"
         },
       },
     ]
   }
   ```

6. **Deploy Test Pod**

   Launch a pod that emits simulated error logs. This should trigger the alert if everything is set up correctly.

   ```hcl
   resource "kubernetes_namespace" "example" {
     metadata {
       name = "example"
     }
   }

   resource "kubernetes_pod" "logger" {
     metadata {
       name      = "logger"
       namespace = kubernetes_namespace.example.metadata[0].name
       labels = {
         app = "logger"
       }
     }

     spec {
       container {
         name  = "logger"
         image = "bash"
         command = [
           "bash",
           "-c",
           <<EOF
           while true; do
             sleep $(shuf -i 1-3 -n 1)
             echo "ERROR: $(date) - Simulated error message $(shuf -i 1-100 -n 1)" 1>&2
           done
           EOF
         ]
       }
     }
   }
   ```