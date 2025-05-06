---
page_title: "Using STACKIT CDN with your own domain"
---
# Using STACKIT CDN with your own domain

## Overview

This guide outlines the process of creating a STACKIT CDN distribution and configuring it to make use of an existing domain using STACKIT DNS.

## Steps

1. **Create a STACKIT CDN and DNS Zone**
    
    Create the CDN distribution and the DNS zone.
    
    ```terraform
    resource "stackit_cdn_distribution" "example_distribution" {
      project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      config = {
        backend = {
          type       = "http"
          origin_url = "mybackend.onstackit.cloud"
        }
        regions = ["EU", "US", "ASIA", "AF", "SA"]
      }
    }
    
    resource "stackit_dns_zone" "example_zone" {
      project_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      name          = "My DNS zone"
      dns_name      = "myapp.runs.onstackit.cloud"
      contact_email = "aa@bb.ccc"
      type          = "primary"
    }
    ```
    
2. **Add CNAME record to your DNS zone**

    If you want to redirect your entire domain to the CDN, you can instead use an A record.
    ```terraform
    resource "stackit_dns_record_set" "example" {
      project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      zone_id    = stackit_dns_zone.example_zone.zone_id
      name       = "cdn"
      type       = "CNAME"
      records    = ["${stackit_cdn_distribution.domains[0].name}."]
    }
    ```
    
3. **Create a STACKIT CDN Custom Domain**
    ```terraform
    # Create a CDN custom domain
    resource "stackit_cdn_custom_domain" "example" {
      project_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      distribution_id = stackit_cdn_distribution.example_distribution.distribution_id
      name            = "${stackit_dns_record_set.example.name}.${stackit_dns_zone.example_zone.dns_name}"
    }
    ```
    
    Now, you can access your content on the url `cdn.myapp.runs.onstackit.cloud`.
