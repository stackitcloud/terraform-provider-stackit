variable "connection_display_name" {}
variable "tunnel1_remote_address" {}
variable "tunnel1_psk" {}
variable "tunnel1_psk_version" {}
variable "tunnel1_bgp_remote_asn" {}
variable "tunnel2_remote_address" {}
variable "tunnel2_psk" {}
variable "tunnel2_psk_version" {}
variable "tunnel2_bgp_remote_asn" {}
variable "remote_subnet" {}
variable "local_subnet" {}
variable "tunnel1_local_peering" {}
variable "tunnel1_remote_peering" {}
variable "tunnel2_local_peering" {}
variable "tunnel2_remote_peering" {}

resource "stackit_vpn_connection" "connection" {
  project_id   = stackit_vpn_gateway.gateway.project_id
  region       = stackit_vpn_gateway.gateway.region
  gateway_id   = stackit_vpn_gateway.gateway.gateway_id
  display_name = var.connection_display_name

  remote_subnet = [var.remote_subnet]
  local_subnet  = [var.local_subnet]

  tunnel1 = {
    remote_address = var.tunnel1_remote_address
    # in the MIN test we use the legacy field, in the MAX test the write-only field
    pre_shared_key_wo         = var.tunnel1_psk
    pre_shared_key_wo_version = var.tunnel1_psk_version

    phase1 = {
      dh_groups             = ["modp2048", "ecp256"]
      encryption_algorithms = ["aes256", "aes128gcm16"]
      integrity_algorithms  = ["sha2_256", "sha2_384"]
      rekey_time            = 25920
    }

    phase2 = {
      dh_groups             = ["modp2048", "ecp256"]
      encryption_algorithms = ["aes256", "aes128gcm16"]
      integrity_algorithms  = ["sha2_256", "sha2_384"]
      rekey_time            = 3240
      start_action          = "start"
    }

    peering = {
      local_address  = var.tunnel1_local_peering
      remote_address = var.tunnel1_remote_peering
    }

    bgp = {
      remote_asn = var.tunnel1_bgp_remote_asn
    }
  }

  tunnel2 = {
    remote_address = var.tunnel2_remote_address
    # in the MIN test we use the legacy field, in the MAX test the write-only field
    pre_shared_key_wo         = var.tunnel2_psk
    pre_shared_key_wo_version = var.tunnel2_psk_version

    phase1 = {
      dh_groups             = ["modp2048", "ecp256"]
      encryption_algorithms = ["aes256", "aes128gcm16"]
      integrity_algorithms  = ["sha2_256", "sha2_384"]
      rekey_time            = 25920
    }

    phase2 = {
      dh_groups             = ["modp2048", "ecp256"]
      encryption_algorithms = ["aes256", "aes128gcm16"]
      integrity_algorithms  = ["sha2_256", "sha2_384"]
      rekey_time            = 3240
      start_action          = "start"
    }

    peering = {
      local_address  = var.tunnel2_local_peering
      remote_address = var.tunnel2_remote_peering
    }

    bgp = {
      remote_asn = var.tunnel2_bgp_remote_asn
    }
  }
}
