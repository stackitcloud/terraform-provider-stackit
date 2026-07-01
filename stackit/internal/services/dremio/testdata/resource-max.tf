
variable "project_id" {}
variable "region" {}

// Instance Variables
variable "display_name" {}
variable "description" {}

variable "authentication_type" {}

variable "authentication_oauth_authority_url" {}
variable "authentication_oauth_client_id" {}
variable "authentication_oauth_client_secret" {}
variable "authentication_oauth_client_jwt_claims_user_name" {}
variable "authentication_oauth_scope" {}
variable "authentication_oauth_parameter_name" {}
variable "authentication_oauth_parameter_value" {}

variable "authentication_type_azuread" { default = null }
variable "authentication_azuread_authority_url" { default = null }
variable "authentication_azuread_client_id" { default = null }
variable "authentication_azuread_client_secret" { default = null }


// User Variables
variable "email" {}
variable "user_description" {}
variable "first_name" {}
variable "last_name" {}
variable "name" {}
variable "password" {}

// Instance Resources
resource "stackit_dremio_instance" "example" {
  project_id   = var.project_id
  region       = var.region
  display_name = var.display_name
  description  = var.description
  authentication = {
    type = var.authentication_type

    oauth = var.authentication_type == "oauth" ? {
      authority_url = var.authentication_oauth_authority_url
      client_id     = var.authentication_oauth_client_id
      client_secret = var.authentication_oauth_client_secret
      jwt_claims = {
        user_name = var.authentication_oauth_client_jwt_claims_user_name
      }
      scope = var.authentication_oauth_scope
      parameters = [
        {
          "name" : var.authentication_oauth_parameter_name,
          "value" : var.authentication_oauth_parameter_value
        }
      ]
    } : null
    azuread = var.authentication_type == "azuread" ? {
      authority_url = var.authentication_azuread_authority_url
      client_id     = var.authentication_azuread_client_id
      client_secret = var.authentication_azuread_client_secret
    } : null
  }
}

data "stackit_dremio_instance" "example" {
  project_id  = var.project_id
  region      = var.region
  instance_id = stackit_dremio_instance.example.instance_id
}

// User Resources
resource "stackit_dremio_user" "example" {
  project_id  = var.project_id
  region      = var.region
  instance_id = stackit_dremio_instance.example.instance_id

  email       = var.email
  description = var.user_description
  first_name  = var.first_name
  last_name   = var.last_name
  name        = var.name
  password    = var.password
}

data "stackit_dremio_user" "example" {
  project_id  = var.project_id
  region      = var.region
  instance_id = stackit_dremio_instance.example.instance_id
  user_id     = stackit_dremio_user.example.user_id
}