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

# Only use the import statement, if you want to import an existing cdn distribution
import {
  to = stackit_cdn_distribution.import-example
  id = "${var.project_id},${var.distribution_id}"
}