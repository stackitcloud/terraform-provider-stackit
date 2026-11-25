resource "stackit_identity_group_members" "members" {
  organization_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  group_id        = stackit_identity_group.members.group_id

  members = [
    {
      id   = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" # User ID
      type = "user"
    },
    {
      id   = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" # Service Account ID
      type = "service_account"
    }
  ]
}
