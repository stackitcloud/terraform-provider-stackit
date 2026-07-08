resource "stackit_resourcemanager_project" "example" {
  name        = "example_project"
  owner_email = "foo.bar@stackit.cloud"
  # in this case a folder or a org-id
  parent_container_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "stackit_authorization_project_role_assignment" "pra" {
  resource_id = stackit_resourcemanager_project.example.project_id
  role        = "reader"
  subject     = "foo.bar@stackit.cloud"
}