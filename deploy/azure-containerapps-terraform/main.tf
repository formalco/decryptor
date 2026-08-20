terraform {
  required_version = ">= 1.1.8"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }
}

provider "azurerm" {
  features {}
  subscription_id = var.subscription_id
}

data "azurerm_key_vault_key" "logs" {
  name         = var.key_name
  key_vault_id = var.key_vault_id
}

# ID is /subscriptions/S/resourceGroups/RG/providers/Microsoft.ContainerRegistry/registries/NAME.
data "azurerm_container_registry" "acr" {
  count               = var.container_registry_id == "" ? 0 : 1
  name                = element(split("/", var.container_registry_id), 8)
  resource_group_name = element(split("/", var.container_registry_id), 4)
}

# Dedicated identity for the decryptor, scoped to decrypt one key (below).
resource "azurerm_user_assigned_identity" "decryptor" {
  name                = var.app_name
  resource_group_name = var.resource_group_name
  location            = var.location
}

# The key to decrypt with comes from the caller-supplied JWE, so scope the
# identity to this one key rather than granting vault-wide decrypt.
resource "azurerm_role_assignment" "crypto_user" {
  scope                = data.azurerm_key_vault_key.logs.resource_versionless_id
  role_definition_name = "Key Vault Crypto User"
  principal_id         = azurerm_user_assigned_identity.decryptor.principal_id
}

resource "azurerm_role_assignment" "acr_pull" {
  count                = var.container_registry_id == "" ? 0 : 1
  scope                = var.container_registry_id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_user_assigned_identity.decryptor.principal_id
}

resource "azurerm_log_analytics_workspace" "decryptor" {
  name                = var.app_name
  resource_group_name = var.resource_group_name
  location            = var.location
  sku                 = "PerGB2018"
  retention_in_days   = var.log_retention_days
}

resource "azurerm_container_app_environment" "decryptor" {
  name                       = var.app_name
  resource_group_name        = var.resource_group_name
  location                   = var.location
  log_analytics_workspace_id = azurerm_log_analytics_workspace.decryptor.id
}

resource "azurerm_container_app" "decryptor" {
  name                         = var.app_name
  resource_group_name          = var.resource_group_name
  container_app_environment_id = azurerm_container_app_environment.decryptor.id
  revision_mode                = "Single"

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.decryptor.id]
  }

  dynamic "registry" {
    for_each = data.azurerm_container_registry.acr
    content {
      server   = registry.value.login_server
      identity = azurerm_user_assigned_identity.decryptor.id
    }
  }

  template {
    min_replicas = 1
    max_replicas = 3

    container {
      name   = "decryptor"
      image  = var.image
      cpu    = 0.25
      memory = "0.5Gi"

      # DefaultAzureCredential needs the client ID to pick this identity.
      env {
        name  = "AZURE_CLIENT_ID"
        value = azurerm_user_assigned_identity.decryptor.client_id
      }
    }
  }

  # The browser calls the decryptor directly, so ingress must be external.
  ingress {
    external_enabled = true
    target_port      = 8080
    transport        = "auto"

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }

    dynamic "ip_security_restriction" {
      for_each = var.allowed_ip_ranges
      content {
        name             = "allowed-${ip_security_restriction.key}"
        ip_address_range = ip_security_restriction.value
        action           = "Allow"
      }
    }
  }

  # The first revision pulls and decrypts, so both grants must land first.
  depends_on = [
    azurerm_role_assignment.crypto_user,
    azurerm_role_assignment.acr_pull,
  ]
}
