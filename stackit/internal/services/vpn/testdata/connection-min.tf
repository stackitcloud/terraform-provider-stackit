variable "connection_display_name" {}
variable "tunnel1_remote_address" {}
variable "tunnel1_psk" {}
variable "tunnel2_remote_address" {}
variable "tunnel2_psk" {}

resource "stackit_vpn_connection" "connection" {
  project_id   = stackit_vpn_gateway.gateway.project_id
  region       = stackit_vpn_gateway.gateway.region
  gateway_id   = stackit_vpn_gateway.gateway.gateway_id
  display_name = var.connection_display_name

  tunnel1 = {
    remote_address = var.tunnel1_remote_address
    # in the MIN test we use the legacy field, in the MAX test the write-only field
    pre_shared_key = var.tunnel1_psk

    phase1 = {
      dh_groups             = ["ecp384"]
      encryption_algorithms = ["aes256"]
      integrity_algorithms  = ["sha2_384"]
    }

    phase2 = {
      dh_groups             = ["ecp384"]
      encryption_algorithms = ["aes256"]
      integrity_algorithms  = ["sha2_384"]
    }
  }

  tunnel2 = {
    remote_address = var.tunnel2_remote_address
    # in the MIN test we use the legacy field, in the MAX test the write-only field
    pre_shared_key = var.tunnel2_psk

    phase1 = {
      dh_groups             = ["ecp384"]
      encryption_algorithms = ["aes256"]
      integrity_algorithms  = ["sha2_384"]
    }

    phase2 = {
      dh_groups             = ["ecp384"]
      encryption_algorithms = ["aes256"]
      integrity_algorithms  = ["sha2_384"]
    }
  }
}
