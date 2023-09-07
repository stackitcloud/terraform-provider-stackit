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
