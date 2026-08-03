resource "time_rotating" "rotate" {
  rotation_days = 30
}

resource "stackit_server" "example" {
  project_id        = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name              = "example"
  machine_type      = "g2i.4"
  availability_zone = "eu01-1"

  boot_volume = {
    source_type           = "image"
    source_id             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    size                  = 32
    delete_on_termination = true
  }

  agent = {
    provisioning_policy = "ALWAYS"
  }

  # Changing this label triggers after_update -> cert is regenerated.
  labels = {
    cert_rotation_id = substr(sha256(time_rotating.rotate.id), 0, 63)
  }

  lifecycle {
    action_trigger {
      events  = [after_update]
      actions = [action.stackit_run_command.renew_cert]
    }
  }
}

action "stackit_run_command" "renew_cert" {
  config {
    project_id            = var.stackit_project_id
    server_id             = stackit_server.example.server_id
    region                = "eu01"
    command_template_name = "RunShellScript"
    parameters = {
      script = <<-EOT
        #!/bin/bash
        set -euo pipefail
        openssl req -x509 -nodes -newkey rsa:2048 -days 90 \
          -subj "/CN=action-server" \
          -keyout /root/server.key \
          -out    /root/server.crt
        echo "renewed at $(date -u +%Y-%m-%dT%H:%M:%SZ)" >> /root/cert.log
        openssl x509 -in /root/server.crt -noout -dates >> /root/cert.log
      EOT
    }
  }
}