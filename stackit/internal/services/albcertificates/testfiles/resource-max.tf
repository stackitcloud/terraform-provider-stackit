
variable "project_id" {}
variable "cert_name" {}
variable "region" {}

resource "tls_private_key" "test" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "tls_self_signed_cert" "test" {
  private_key_pem = tls_private_key.test.private_key_pem

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

resource "stackit_alb_certificate" "certificate" {
  project_id  = var.project_id
  name        = var.cert_name
  region      = var.region
  private_key = tls_private_key.test.private_key_pem
  public_key  = tls_self_signed_cert.test.cert_pem
}
