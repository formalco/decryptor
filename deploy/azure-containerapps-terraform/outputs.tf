output "decryptor_uri" {
  description = "The /decrypt endpoint to set as the Formal encryption key's decryptor URI."
  value       = "https://${azurerm_container_app.decryptor.ingress[0].fqdn}/decrypt"
}

output "identity_principal_id" {
  description = "Object ID of the managed identity the decryptor runs as, to grant key permissions on an access policy vault."
  value       = azurerm_user_assigned_identity.decryptor.principal_id
}
