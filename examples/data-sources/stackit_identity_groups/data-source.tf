data "stackit_identity_groups" "group" {
  organization_id    = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  display_name_regex = ".*-admin-.*"
  sort_ascending     = true
}
