# Create a CDN custom domain
resource "stackit_cdn_custom_domain" "example" {
  project_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  distribution_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "https://xxx.xxx"
}
