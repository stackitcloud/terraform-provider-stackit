resource "stackit_observability_scrapeconfig" "example" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name         = "example-job"
  metrics_path = "/my-metrics"
  saml2 = {
    enable_url_parameters = true
  }
  targets = [
    {
      urls = ["url1", "urls2"]
      labels = {
        "url1" = "dev"
      }
    }
  ]
}

# Only use the import statement, if you want to import an existing observability scrapeconfig
import {
  to = stackit_observability_scrapeconfig.import-example
  id = "${var.project_id},${var.observability_instance_id},${var.observability_scrapeconfig_name}"
}
