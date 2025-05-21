variable "project_id" {}
variable "name" {}
variable "dns_name" {}

variable "record_name" {}
variable "record_record1" {}
variable "record_type" {}


resource "stackit_dns_zone" "zone" {
  project_id = var.project_id
  name       = var.name
  dns_name   = var.dns_name
}


resource "stackit_dns_record_set" "record_set" {
  project_id = var.project_id
  zone_id    = stackit_dns_zone.zone.zone_id
  name       = var.record_name
  records = [
    var.record_record1
  ]
  type = var.record_type
}

data "stackit_dns_zone" "zone" {
  project_id = var.project_id
  zone_id    = stackit_dns_zone.zone.zone_id
}

data "stackit_dns_zone" "zone_name" {
  project_id = var.project_id
  dns_name   = stackit_dns_zone.zone.dns_name
}

data "stackit_dns_record_set" "record_set" {
  project_id    = var.project_id
  zone_id       = stackit_dns_zone.zone.zone_id
  record_set_id = stackit_dns_record_set.record_set.record_set_id
}
