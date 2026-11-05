output "decryptor_uri" {
  description = "Decryptor endpoint registered on the Formal encryption key."
  value       = module.decryptor.decryptor_uri
}

output "encryption_key_id" {
  description = "Formal encryption key ID to set as encryption_key_id on formal_log_schema."
  value       = formal_encryption_key.logs.id
}
