---
page_title: "Using the STACKIT Loadbalancer together with STACKIT Observability"
---
# Using the STACKIT Loadbalancer together with STACKIT Observability

## Overview

This guide explains how to configure the STACKIT Loadbalancer product to send metrics and logs to a STACKIT Observability instance.

1. **Set Up Providers**

   Begin by configuring the STACKIT provider to connect to the STACKIT services.

   ```hcl
   provider "stackit" {
     default_region = "eu01"
   }
   ```

2. **Create an Observability instance**

   Establish a STACKIT Observability instance and its credentials.

   ```hcl
   resource "stackit_observability_instance" "observability01" {
     project_id                             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     name                                   = "example-instance"
     plan_name                              = "Observability-Monitoring-Medium-EU01"
     acl                                    = ["0.0.0.0/0"]
     metrics_retention_days                 = 90
     metrics_retention_days_5m_downsampling = 90
     metrics_retention_days_1h_downsampling = 90
   }

   resource "stackit_observability_credential" "observability01-credential" {
     project_id                             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     instance_id = stackit_observability_instance.observability01.instance_id
   }
   ```

3. **Create STACKIT Loadbalancer credentials reference**

   Create a STACKIT Loadbalancer credentials which will be used in the STACKIT Loadbalancer resource as a reference.

   ```hcl
    resource "stackit_loadbalancer_observability_credential" "example" {
      project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      display_name = "example-credentials"
      username     = stackit_observability_credential.observability01-credential.username
      password     = stackit_observability_credential.observability01-credential.password
    }
   ```

4. **Create the STACKIT Loadbalancer**

   ```hcl
   # Create a network
   resource "stackit_network" "example_network" {
     project_id       = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     name             = "example-network"
     ipv4_nameservers = ["8.8.8.8"]
     ipv4_prefix      = "192.168.0.0/25"
     labels = {
       "key" = "value"
     }
     routed = true
   }

   # Create a network interface
   resource "stackit_network_interface" "nic" {
     project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     network_id = stackit_network.example_network.network_id
   }

   # Create a public IP for the load balancer
   resource "stackit_public_ip" "public-ip" {
     project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     lifecycle {
       ignore_changes = [network_interface_id]
     }
   }

   # Create a key pair for accessing the server instance
   resource "stackit_key_pair" "keypair" {
     name       = "example-key-pair"
     # set the path of your public key file here
     public_key = chomp(file("/home/bob/.ssh/id_ed25519.pub"))
   }

   # Create a server instance
   resource "stackit_server" "boot-from-image" {
     project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     name       = "example-server"
     boot_volume = {
       size        = 64
       source_type = "image"
       source_id   = "59838a89-51b1-4892-b57f-b3caf598ee2f" // Ubuntu 24.04
     }
     availability_zone = "eu01-1"
     machine_type      = "g1.1"
     keypair_name      = stackit_key_pair.keypair.name
     network_interfaces = [
         stackit_network_interface.nic.network_interface_id
     ]
   }

   # Create a load balancer
   resource "stackit_loadbalancer" "example" {
     project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
     name       = "example-load-balancer"
     target_pools = [
       {
         name        = "example-target-pool"
         target_port = 80
         targets = [
           {
             display_name = stackit_server.boot-from-image.name
             ip           = stackit_network_interface.nic.ipv4
           }
         ]
         active_health_check = {
           healthy_threshold   = 10
           interval            = "3s"
           interval_jitter     = "3s"
           timeout             = "3s"
           unhealthy_threshold = 10
         }
       }
     ]
     listeners = [
       {
         display_name = "example-listener"
         port         = 80
         protocol     = "PROTOCOL_TCP"
         target_pool  = "example-target-pool"
       }
     ]
     networks = [
       {
         network_id = stackit_network.example_network.network_id
         role       = "ROLE_LISTENERS_AND_TARGETS"
       }
     ]
     external_address = stackit_public_ip.public-ip.ip
     options = {
       private_network_only = false
       observability = {
   	     logs = {
   	        # uses the load balancer credential from the last step
   	     	credentials_ref = stackit_loadbalancer_observability_credential.example.credentials_ref
   	     	# uses the observability instance from step 1
   	     	push_url = stackit_observability_instance.observability01.logs_push_url
   	     }
   	     metrics = {
   	        # uses the load balancer credential from the last step
   	     	credentials_ref = stackit_loadbalancer_observability_credential.example.credentials_ref
   	     	# uses the observability instance from step 1
   	     	push_url = stackit_observability_instance.observability01.metrics_push_url
   	     }
       }
     }
   }
   ```
