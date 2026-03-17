data "stackit_service_accounts" "ske_sa_suffix" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  email_suffix = "@ske.sa.stackit.cloud"
}

resource "stackit_service_account" "iam" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "kms"
}

resource "stackit_authorization_project_role_assignment" "pr_sa" {
  resource_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  role        = "kms.admin"
  subject     = stackit_service_account.iam.email
}

// Assign the Act-As permissions to the previously created ServiceAccount.
// The SKE ServiceAccount is now authorized to access KMS upon the behalf of stackit_service_account.iam
resource "stackit_authorization_service_account_role_assignment" "sa" {
  resource_id = stackit_service_account.iam.service_account_id
  role        = "user"
  subject     = data.stackit_service_accounts.ske_sa_suffix.items.0.email
}

# Only use the import statement, if you want to import an existing service account assignment
import {
  to = stackit_authorization_service_account_assignment.sa
  id = "${var.resource_id},${var.service_account_assignment_role},${var.service_account_assignment_subject}"
}
