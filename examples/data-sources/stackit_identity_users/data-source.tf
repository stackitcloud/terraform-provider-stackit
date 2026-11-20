data "stackit_identity_users" "users" {
  organization_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  email_regex     = ".*@stackit\\.cloud"
  sort_ascending  = true
}
