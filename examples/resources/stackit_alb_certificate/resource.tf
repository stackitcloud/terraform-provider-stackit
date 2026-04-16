variable "project_id" {
  description = "The STACKIT Project ID"
  type        = string
  default     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

# Create a RAS key pair
resource "tls_private_key" "example" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

# Create a TLS certificate
resource "tls_self_signed_cert" "example" {
  private_key_pem = tls_private_key.example.private_key_pem

  subject {
    common_name  = "localhost"
    organization = "STACKIT Test"
  }

  validity_period_hours = 12

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "server_auth",
  ]
}

# Create a ALB certificate
resource "stackit_alb_certificate" "certificate" {
  project_id  = var.project_id
  name        = "example-certificate"
  private_key = tls_private_key.example.private_key_pem
  public_key  = tls_self_signed_cert.example.cert_pem
}