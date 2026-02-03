resource "stackit_cdn_distribution" "example_distribution" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  config = {
    backend = {
      type       = "http"
      origin_url = "https://mybackend.onstackit.cloud"
      geofencing = {
        "https://mybackend.onstackit.cloud" = ["DE"]
      }
    }
    regions           = ["EU", "US", "ASIA", "AF", "SA"]
    blocked_countries = ["DE", "AT", "CH"]

    optimizer = {
      enabled = true
    }
  }
}

resource "stackit_cdn_distribution" "example_bucket_distribution" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  config = {
    backend = {
      type       = "bucket"
      bucket_url = "https://my-private-bucket.s3.eu-central-1.amazonaws.com"
      region     = "eu01"
      
      # Credentials are required for bucket backends
      # It is strongly recommended to use variables for secrets
      access_key = var.bucket_access_key
      secret_key = var.bucket_secret_key
    }
    regions           = ["EU", "US"]
    blocked_countries = ["CN", "RU"]

    optimizer = {
      enabled = false
    }
  }
}

# Only use the import statement, if you want to import an existing cdn distribution
import {
  to = stackit_cdn_distribution.import-example
  id = "${var.project_id},${var.distribution_id}"
}