provider "stackit" {
  default_region = "eu01"
}

# Authentication

# Workload Identity Federation flow 
provider "stackit" {
  default_region                  = "eu01"
  service_account_email           = var.service_account_email
  service_account_federated_token = var.service_account_federated_token
  use_oidc                        = true
}

# Workload Identity Federation flow (using path)
provider "stackit" {
  default_region                       = "eu01"
  service_account_email                = var.service_account_email
  service_account_federated_token_path = var.service_account_federated_token_path
  use_oidc                             = true
}

# Key flow
provider "stackit" {
  default_region      = "eu01"
  service_account_key = var.service_account_key
  private_key         = var.private_key
}

# Key flow (using path)
provider "stackit" {
  default_region           = "eu01"
  service_account_key_path = var.service_account_key_path
  private_key_path         = var.private_key_path
}

