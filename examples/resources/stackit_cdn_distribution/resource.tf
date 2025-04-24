# Create a CDN distribution
resource "stackit_cdn_distribution" "example_distribution" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  config = {
    backend = {
      type       = "http"
      origin_url = "mybackend.onstackit.cloud"
    }
    regions = ["EN", "US", "ASIA", "AF", "SA"]
  }
}
