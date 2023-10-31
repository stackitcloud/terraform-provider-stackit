resource "stackit_loadbalancer" "example" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "example-load-balancer"
  target_pools = [
    {
      name        = "example-target-pool"
      target_port = 80
      targets = [
        {
          display_name = "example-target"
          ip           = openstack_compute_instance_v2.example.network.0.fixed_ip_v4
        }
      ]
      active_health_check = {
        healthy_threshold   = 10
        interval            = "3s"
        interval_jitter     = "3s"
        timeout             = "3s"
        unhealthy_threshold = 10
      }
    }
  ]
  listeners = [
    {
      display_name = "example-listener"
      port         = 80
      protocol     = "PROTOCOL_TCP"
      target_pool  = "example-target-pool"
    }
  ]
  networks = [
    {
      network_id = openstack_networking_network_v2.example.id
      role       = "ROLE_LISTENERS_AND_TARGETS"
    }
  ]
  external_address = openstack_networking_floatingip_v2.example.address
  options = {
    private_network_only = false
  }
}