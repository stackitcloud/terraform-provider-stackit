---
page_title: "Alerting with Kube-State-Metrics in STACKIT Observability"
---
# Alerting with Kube-State-Metrics in STACKIT Observability

## Overview

This guide explains how to configure the STACKIT Observability product to send alerts using metrics gathered from kube-state-metrics.

1. **Set Up Providers**

   Begin by configuring the STACKIT and Kubernetes providers to connect to the STACKIT services.

   ```hcl
   provider "stackit" {
     default_region = "eu01"
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
     project_id             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     name                   = "example"
     kubernetes_version_min = "1.31"
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

4. **Install Prometheus Operator**

   Use the Prometheus Helm chart to install kube-state-metrics and transfer metrics to the STACKIT Observability instance. Customize the helm values as needed for your deployment.

   ```yaml
   # helm values
   # save as prom-values.tftpl
   prometheus:
     enabled: true
     agentMode: true
     prometheusSpec:
       enableRemoteWriteReceiver: true
       scrapeInterval: 60s
       evaluationInterval: 60s
       replicas: 1
       storageSpec:
         volumeClaimTemplate:
           spec:
             storageClassName: premium-perf4-stackit
             accessModes: ['ReadWriteOnce']
             resources:
               requests:
                 storage: 80Gi
       remoteWrite:
         - url: ${metrics_push_url}
           queueConfig:
             batchSendDeadline: '5s'
             # both values need to be configured according to your observability plan
             capacity: 30000
             maxSamplesPerSend: 3000
             writeRelabelConfigs:
               - sourceLabels: ['__name__']
                 regex: 'apiserver_.*|etcd_.*|prober_.*|storage_.*|workqueue_(work|queue)_duration_seconds_bucket|kube_pod_tolerations|kubelet_.*|kubernetes_feature_enabled|instance_scrape_target_status'
                 action: 'drop'
               - sourceLabels: ['namespace']
                 regex: 'example'
                 action: 'keep'
           basicAuth:
             username:
               key: username
               name: ${secret_name}
             password:
               key: password
               name: ${secret_name}

   grafana:
     enabled: false

   defaultRules:
     create: false

   alertmanager:
     enabled: false

   nodeExporter:
     enabled: true

   kube-state-metrics:
     enabled: true
     customResourceState:
       enabled: true
     collectors:
       - deployments
       - pods
   ```

   ```hcl
   resource "kubernetes_namespace" "monitoring" {
     metadata {
       name = "monitoring"
     }
   }

   resource "kubernetes_secret" "argus_prometheus_authorization" {
     metadata {
       name      = "argus-prometheus-credentials"
       namespace = kubernetes_namespace.monitoring.metadata[0].name
     }

     data = {
       username = stackit_observability_credential.example.username
       password = stackit_observability_credential.example.password
     }
   }

   resource "helm_release" "prometheus_operator" {
     name       = "prometheus-operator"
     repository = "https://prometheus-community.github.io/helm-charts"
     chart      = "kube-prometheus-stack"
     version    = "60.1.0"
     namespace  = kubernetes_namespace.monitoring.metadata[0].name

     values = [
       templatefile("prom-values.tftpl", {
         metrics_push_url = stackit_observability_instance.example.metrics_push_url
         secret_name      = kubernetes_secret.argus_prometheus_authorization.metadata[0].name
       })
     ]
   }
   ```

5. **Create Alert Group**

   Define an alert group with a rule to notify when a pod is running in the "example" namespace.

   ```hcl
   resource "stackit_observability_alertgroup" "example" {
     project_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     instance_id = stackit_observability_instance.example.instance_id
     name        = "TestAlertGroup"
     interval    = "2h"
     rules = [
       {
         alert      = "SimplePodCheck"
         expression = "sum(kube_pod_status_phase{phase=\"Running\", namespace=\"example\"}) > 0"
         for        = "60s"
         labels = {
           severity = "critical"
         },
         annotations = {
           summary     = "Test Alert is working"
           description = "Test Alert"
         }
       },
     ]
   }
   ```

6. **Deploy Test Pod**

   Deploy a test pod; doing so should trigger an email notification, as the deployment satisfies the conditions defined in the alert group rule. In a real-world scenario, you would typically configure alerts to monitor pods for error states instead.

   ```hcl
   resource "kubernetes_namespace" "example" {
     metadata {
       name = "example"
     }
   }

   resource "kubernetes_pod" "example" {
     metadata {
       name      = "nginx"
       namespace = kubernetes_namespace.example.metadata[0].name
       labels = {
         app = "nginx"
       }
     }

     spec {
       container {
         image = "nginx:latest"
         name  = "nginx"
       }
     }
   }
   ```