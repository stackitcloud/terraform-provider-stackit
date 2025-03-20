resource "stackit_argus_scrapeconfig" "example" {
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

### Example move

#	Example to move the deprecated `stackit_argus_scrapeconfig` resource to the new `stackit_observability_scrapeconfig` resource:
#	1. Add a new `stackit_observability_scrapeconfig` resource with the same values like your previous `stackit_argus_scrapeconfig` resource.
#	2. Add a moved block which reference the `stackit_argus_scrapeconfig` and `stackit_observability_scrapeconfig` resource.
#	3. Remove your old `stackit_argus_scrapeconfig` resource and run `$ terraform apply`.

resource "stackit_argus_scrapeconfig" "example" {
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

moved {
  from = stackit_argus_scrapeconfig.example
  to   = stackit_observability_scrapeconfig.example
}

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
