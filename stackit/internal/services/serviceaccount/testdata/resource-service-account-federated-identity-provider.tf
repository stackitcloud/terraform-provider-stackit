variable "project_id" {
  type = string
}

variable "provider_name" {
  type = string
}

variable "sub" {
  type = string
}

resource "stackit_service_account" "sa" {
  project_id = var.project_id
  name       = "test-sa"
}

resource "stackit_service_account_federated_identity_provider" "provider" {
  project_id            = stackit_service_account.sa.project_id
  service_account_email = stackit_service_account.sa.email
  name                  = var.provider_name
  issuer                = "https://accounts.stackit.cloud"

  assertions = [
    {
      item     = "iss"
      operator = "equals"
      value    = "https://accounts.stackit.cloud"
    },
    {
      item     = "sub"
      operator = "equals"
      value    = var.sub
    },
    {
      item     = "aud"
      operator = "equals"
      value    = "sts.accounts.stackit.cloud"
    }
  ]
}
