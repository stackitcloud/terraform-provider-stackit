provider "stackit" {
  experiments = ["ske"]
}

resource "stackit_ske_cluster" "example" {
  # ... cluster configuration ...
}

# We use the cluster ID ternary to force evaluation during the Apply phase.
# Unlike managed resources, ephemeral resources evaluate during the Plan phase
# if inputs are known, which would trigger a 404 before the cluster exists.
ephemeral "stackit_ske_kubeconfig" "example" {
  project_id   = stackit_ske_cluster.example.project_id
  cluster_name = stackit_ske_cluster.example.id != "" ? stackit_ske_cluster.example.name : ""
}

provider "kubernetes" {
  host                   = yamldecode(ephemeral.stackit_ske_kubeconfig.example.kube_config).clusters.0.cluster.server
  client_certificate     = base64decode(yamldecode(ephemeral.stackit_ske_kubeconfig.example.kube_config).users.0.user.client-certificate-data)
  client_key             = base64decode(yamldecode(ephemeral.stackit_ske_kubeconfig.example.kube_config).users.0.user.client-key-data)
  cluster_ca_certificate = base64decode(yamldecode(ephemeral.stackit_ske_kubeconfig.example.kube_config).clusters.0.cluster.certificate-authority-data)
}
