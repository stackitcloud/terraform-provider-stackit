# custom domain
resource "stackit_cdn_custom_domain" "custom_domain" {
  project_id      = var.project_id
  distribution_id = stackit_cdn_distribution.distribution.distribution_id
  name            = "${stackit_dns_record_set.dns_record.name}.${stackit_dns_zone.dns_zone.dns_name}"
  certificate = {
    certificate = var.certificate
    private_key = var.private_key
  }
}

data "stackit_cdn_custom_domain" "custom_domain" {
  project_id      = var.project_id
  distribution_id = stackit_cdn_distribution.distribution.distribution_id
  name            = "${stackit_dns_record_set.dns_record.name}.${stackit_dns_zone.dns_zone.dns_name}"
  depends_on      = [stackit_cdn_custom_domain.custom_domain]
}