resource "stackit_authorization_organization_custom_role" "example" {
  resource_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name        = "my.custom.role"
  description = "Some description"
  permissions = [
    "iam.subject.get"
  ]
}

# Only use the import statement, if you want to import an existing custom role
import {
  to = stackit_authorization_organization_custom_role.import-example
  id = "${var.organization_id},${var.custom_role_id}"
}

