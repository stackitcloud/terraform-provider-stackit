resource "stackit_kms_wrapping_key" "example" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  keyring_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  display_name = "example-name"
  protection   = "software"
  algorithm    = "rsa_2048_oaep_sha256"
  purpose      = "wrap_symmetric_key"
}
