resource "stackit_resourcemanager_project" "example" {
  name        = "example_project"
  owner_email = "foo.bar@stackit.cloud"
  # in this case a folder or a org-id
  parent_container_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "stackit_authorization_project_role_assignment" "pra" {
  resource_id = stackit_resourcemanager_project.example.folder_id
  role        = "reader"
  subject     = "foo.bar@stackit.cloud"
}

# Only use the import statement, if you want to import an existing project role assignment
import {
  to = stackit_authorization_project_role_assignment.import-example
  id = "${var.project_id},${var.project_role_assignment_role},${var.project_role_assignment_subject}"
}
