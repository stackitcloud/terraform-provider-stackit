resource "stackit_cdn_custom_domain" "example" {
  project_id      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  distribution_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name            = "https://xxx.xxx"
  certificate = {
    certificate = "-----BEGIN CERTIFICATE-----\nY2VydGlmaWNhdGVfZGF0YQ==\n-----END CERTIFICATE---"
    private_key = "-----BEGIN RSA PRIVATE KEY-----\nY2VydGlmaWNhdGVfZGF0YQ==\n-----END RSA PRIVATE KEY---"
  }
}