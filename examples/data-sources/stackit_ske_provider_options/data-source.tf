data "stackit_ske_provider_options" "default" {}

data "stackit_ske_provider_options" "eu02" {
  region = "eu02"
}

locals {
  k8s_versions = [
    for v in data.stackit_ske_provider_options.default.kubernetes_versions :
    v.version if v.state == "supported"
  ]
  first_k8s_version = length(local.k8s_versions) > 0 ? local.k8s_versions[0] : ""
  last_k8s_version  = length(local.k8s_versions) > 0 ? local.k8s_versions[length(local.k8s_versions) - 1] : ""


  flatcar_supported_versions = flatten([
    for mi in data.stackit_ske_provider_options.default.machine_images : [
      for v in mi.versions :
      v.version if mi.name == "flatcar" && v.state == "supported"
    ]
  ])

  ubuntu_supported_versions = flatten([
    for mi in data.stackit_ske_provider_options.default.machine_images : [
      for v in mi.versions :
      v.version if mi.name == "ubuntu" && v.state == "supported"
    ]
  ])
}