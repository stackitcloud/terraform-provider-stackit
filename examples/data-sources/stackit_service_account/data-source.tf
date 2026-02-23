data "stackit_service_account" "sa_exact" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  email      = "foo-vshp19@sa.stackit.cloud"
}

// Querying the SKE service account using a regular expression for the email
data "stackit_service_account" "sa_ske" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  # This regex matches any standard email prefix ending exactly with the SKE domain.
  email_regex = ".*@ske\\.sa\\.stackit\\.cloud$"
}