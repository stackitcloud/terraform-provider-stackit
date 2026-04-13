variable "project_id" {}
variable "regions" {}
variable "backend_http_type" {}
variable "backend_origin_url" {}
variable "geofencing_list" {}
variable "blocked_countries" {}
variable "optimizer" {}
variable "origin_request_headers_name" {}
variable "origin_request_headers_value" {}
variable "certificate" {}
variable "private_key" {}
variable "redirect_target_url" {}
variable "redirect_status_code" {}
variable "redirect_rule_description" {}
variable "redirect_rule_enabled" {}
variable "redirect_rule_match_condition" {}
variable "redirect_matcher_value" {}
variable "redirect_matcher_condition" {}

# dns
variable "dns_zone_name" {}
variable "dns_name" {}
variable "dns_record_name" {}

resource "stackit_dns_zone" "dns_zone" {
  project_id    = var.project_id
  name          = var.dns_zone_name
  dns_name      = var.dns_name
  contact_email = "aa@bb.cc"
  type          = "primary"
  default_ttl   = 3600
}

resource "stackit_dns_record_set" "dns_record" {
  project_id = var.project_id
  zone_id    = stackit_dns_zone.dns_zone.zone_id
  name       = var.dns_record_name
  type       = "CNAME"
  records    = ["${stackit_cdn_distribution.distribution.domains[0].name}."]
}

resource "stackit_cdn_distribution" "distribution" {
  project_id = var.project_id
  config = {
    regions = var.regions
    optimizer = {
      enabled = var.optimizer
    }
    redirects = {
      rules = [
        {
          description          = var.redirect_rule_description
          enabled              = var.redirect_rule_enabled
          target_url           = var.redirect_target_url
          status_code          = var.redirect_status_code
          rule_match_condition = var.redirect_rule_match_condition
          matchers = [
            {
              values                = [var.redirect_matcher_value]
              value_match_condition = var.redirect_matcher_condition
            }
          ]
        }
      ]
    }
    backend = {
      type       = var.backend_http_type
      origin_url = var.backend_origin_url
      origin_request_headers = {
        (var.origin_request_headers_name) = var.origin_request_headers_value
      }
      geofencing = {
        (var.backend_origin_url) = var.geofencing_list
      }
    }
    blocked_countries = var.blocked_countries
  }
}

data "stackit_cdn_distribution" "distribution" {
  project_id      = var.project_id
  distribution_id = stackit_cdn_distribution.distribution.distribution_id
}