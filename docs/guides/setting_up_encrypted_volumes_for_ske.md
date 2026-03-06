---
page_title: "Setting up Encrypted Volumes for STACKIT Kubernetes Engine (SKE)"
---

# Setting up Encrypted Volumes for STACKIT Kubernetes Engine (SKE)

~> This guide assumes that your project or organization has been enabled for a preview version of the STACKIT CSI Driver. If you wish to use encrypted volumes, please contact your account manager.

## Overview

This guide demonstrates how to roll out an encrypted storage class for SKE using the STACKIT Key Management Service (KMS). To achieve this, we use a **Service Account Impersonation** (Act-As) pattern. This allows the internal SKE service account to perform encryption and decryption tasks on behalf of a user-managed service account that has been granted access to your KMS keys.

## Steps

### 1. Configure the SKE Cluster

Create a standard SKE cluster. We also generate a kubeconfig dynamically to allow the `kubernetes` provider to interact with the cluster within the same Terraform execution.

```hcl
resource "stackit_ske_cluster" "default" {
  project_id             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name                   = "ske-enc-vol"
  kubernetes_version_min = "1.33"

  node_pools = [{
    name               = "standard"
    machine_type       = "c2i.4"
    minimum            = 1
    maximum            = 3
    availability_zones = ["eu01-1"]
    os_name            = "flatcar"
    volume_size        = 32
  }]
}

resource "stackit_ske_kubeconfig" "default" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  cluster_name = stackit_ske_cluster.default.name
  refresh      = true
}
```


### 2. Identify the Internal SKE Service Account

Each STACKIT project with a SKE Cluster deployed has a dedicated, internal service account used by SKE. We need to look this up to grant it permissions in a later step.

```hcl
data "stackit_service_accounts" "ske_internal" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  email_suffix = "@ske.sa.stackit.cloud"

  depends_on = [stackit_ske_cluster.default]
}
```

### 3. Setup KMS Infrastructure

Define the Keyring and the specific Key that will be used to encrypt the block storage volumes.

```hcl
resource "stackit_kms_keyring" "encryption" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  display_name = "ske-volume-keyring"
}

resource "stackit_kms_key" "volume_key" {
  project_id   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  keyring_id   = stackit_kms_keyring.encryption.keyring_id
  display_name = "volume-encryption-key"
  protection   = "software"
  algorithm    = "aes_256_gcm"
  purpose      = "symmetric_encrypt_decrypt"
}
```

### 4. Configure Identity and Permissions (Act-As)

This is the most critical part of the setup. We create a **manager** service account that holds the KMS permissions, and then authorize the SKE internal service account to **Act-As** (impersonate) that manager.

```hcl
# Create the service account that 'owns' the KMS access
resource "stackit_service_account" "kms_manager" {
  project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name       = "volume-encryptor"
}

# Grant the 'kms.admin' role to the manager service-account
resource "stackit_authorization_project_role_assignment" "kms_user" {
  resource_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  role        = "kms.admin"
  subject     = stackit_service_account.kms_manager.email
}

# Authorize the internal SKE account to impersonate the kms manager service-account (Act-As)
resource "stackit_authorization_service_account_role_assignment" "ske_impersonation" {
  resource_id = stackit_service_account.kms_manager.service_account_id
  role        = "user"
  subject     = data.stackit_service_accounts.ske_internal.items[0].email
}
```

### 5. Create the Encrypted Storage Class in Kubernetes

Define the `kubernetes_storage_class`. We pass the IDs of the KMS resources and the email of our manager service account into the parameters.

```hcl
resource "kubernetes_storage_class" "encrypted_premium" {
  metadata {
    name = "stackit-encrypted-premium"
  }

  storage_provisioner    = "block-storage.csi.stackit.cloud"
  reclaim_policy         = "Delete"
  allow_volume_expansion = true
  volume_binding_mode    = "WaitForFirstConsumer"

  parameters = {
    type              = "storage_premium_perf6"
    encrypted         = "true"
    kmsKeyID          = stackit_kms_key.volume_key.key_id
    kmsKeyringID      = stackit_kms_keyring.encryption.keyring_id
    kmsProjectID      = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    kmsKeyVersion     = "1"
    kmsServiceAccount = stackit_service_account.kms_manager.email
  }

  depends_on = [
    stackit_authorization_service_account_role_assignment.ske_impersonation,
    stackit_authorization_project_role_assignment.kms_user
  ]
}
```

### 6. Verify with a Persistent Volume Claim (PVC)

You can now create a PVC using the new storage class. When a pod claims this volume, the STACKIT CSI driver will automatically use the KMS key to provide an encrypted volume.

```hcl
resource "kubernetes_persistent_volume_claim" "test_pvc" {
  metadata {
    name = "test-encryption-pvc"
  }

  spec {
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = "10Gi"
      }
    }

    storage_class_name = kubernetes_storage_class.encrypted_premium.metadata[0].name
  }
}
```

### 7. Create a Pod to Consume the Volume

```hcl
resource "kubernetes_pod" "test_app" {
  metadata {
    name = "encrypted-volume-test"
  }

  spec {
    container {
      image = "nginx:latest"
      name  = "web-server"

      volume_mount {
        mount_path = "/usr/share/nginx/html"
        name       = "data-volume"
      }
    }

    volume {
      name = "data-volume"
      persistent_volume_claim {
        claim_name = "test-encryption-pvc"
      }
    }
  }
}
```