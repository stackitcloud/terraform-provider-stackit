variable "project_id" {}
variable "name" {}
variable "dns_name" {}
variable "acl" {}
variable "active" {}
variable "contact_email" {}
variable "default_ttl" {}
variable "description" {}
variable "expire_time" {}
variable "is_reverse_zone" {}
# variable "negative_cache" {}
variable "primaries" {}
variable "refresh_time" {}
variable "retry_time" {}
variable "type" {}

variable "record_name" {}
variable "record_record1" {}
variable "record_active" {}
variable "record_comment" {}
variable "record_ttl" {}
variable "record_type" {}




resource "stackit_dns_zone" "zone" {
  project_id      = var.project_id
  name            = var.name
  dns_name        = var.dns_name
  acl             = var.acl
  active          = var.active
  contact_email   = var.contact_email
  default_ttl     = var.default_ttl
  description     = var.description
  expire_time     = var.expire_time
  is_reverse_zone = var.is_reverse_zone
  # negative_cache  = var.negative_cache
  primaries       = var.primaries
  refresh_time    = var.refresh_time
  retry_time      = var.retry_time
  type            = var.type
}


resource "stackit_dns_record_set" "record_set" {
  project_id = var.project_id
  zone_id    = stackit_dns_zone.zone.zone_id
  name       = var.record_name
  records = [
    var.record_record1
  ]

  active  = var.record_active
  comment = var.record_comment
  ttl     = var.record_ttl
  type    = var.record_type
}

data "stackit_dns_zone" "zone" {
  project_id = var.project_id
  zone_id    = stackit_dns_zone.zone.zone_id
}

data "stackit_dns_record_set" "record_set" {
  project_id    = var.project_id
  zone_id       = stackit_dns_zone.zone.zone_id
  record_set_id = stackit_dns_record_set.record_set.record_set_id
}
