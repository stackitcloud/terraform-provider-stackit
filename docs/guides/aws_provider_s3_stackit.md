---
page_title: "Using AWS Provider for STACKIT Object Storage (S3 compatible)"
---
# Using AWS Provider for STACKIT Object Storage (S3 compatible)

## Overview

This guide outlines the process of utilizing the [AWS Terraform Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs) alongside the STACKIT provider to create and manage STACKIT Object Storage (S3 compatible) ressources.

## Steps

1. **Configure STACKIT Provider**

    First, configure the STACKIT provider to connect to the STACKIT services.

    ```hcl
    provider "stackit" {
      region = "eu01"
    }
    ```

2. **Define STACKIT Object Storage Bucket**

    Create a STACKIT Object Storage Bucket and obtain credentials for the AWS provider.

    ```hcl
    resource "stackit_objectstorage_bucket" "example" {
      project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      name       = "example"
    }

    resource "stackit_objectstorage_credentials_group" "example" {
      project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      name       = "example-credentials-group"
   }

   resource "stackit_objectstorage_credential" "example" {
      project_id           = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      credentials_group_id = stackit_objectstorage_credentials_group.example.credentials_group_id
      expiration_timestamp = "2027-01-02T03:04:05Z"
   }
    ```

3. **Configure AWS Provider**

   Configure the AWS Provider to connect to the STACKIT Object Storage bucket.

    ```hcl
    provider "aws" {
      region                      = "eu01"
      skip_credentials_validation = true
      skip_region_validation      = true
      skip_requesting_account_id  = true

      access_key                  = stackit_objectstorage_credential.example.access_key
      secret_key                  = stackit_objectstorage_credential.example.secret_access_key

      endpoints {
         s3 = "https://object.storage.eu01.onstackit.cloud"
      }
   }
    ```

4. **Use the provider to manage objects or policies**

    ```hcl
      resource "aws_s3_object" "test_file" {
         bucket = stackit_objectstorage_bucket.example.name
         key    = "hello_world.txt"
         source = "files/hello_world.txt"
         content_type = "text/plain"
         etag = filemd5("files/hello_world.txt")
      }

      resource "aws_s3_bucket_policy" "allow_public_read_access" {
         bucket = stackit_objectstorage_bucket.test20.name
         policy = <<EOF
         {
            "Statement":[
               {
               "Sid": "Public GET",
               "Effect":"Allow",
               "Principal":"*",
               "Action":"s3:GetObject",
               "Resource":"urn:sgws:s3:::example/*"
               }
            ]
         }
         EOF
      }
    ```