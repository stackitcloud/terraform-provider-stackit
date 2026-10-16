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

resource "stackit_observability_scrapeconfig" "example-bearer-token" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  instance_id  = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name         = "example-job-bearer-token"
  metrics_path = "/metrics"
  scheme       = "https"
  bearer_token = var.scrape_bearer_token
  honor_labels = true
  tls_config = {
    insecure_skip_verify = true
  }
  targets = [
    {
      urls = ["metrics.example.com:443"]
    }
  ]
}
