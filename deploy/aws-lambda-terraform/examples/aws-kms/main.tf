terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 6.0"
    }
    formal = {
      source  = "formalco/formal"
      version = ">= 4.30.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# Reads FORMAL_API_KEY from the environment.
provider "formal" {}

resource "aws_kms_key" "formal_logs" {
  description              = "Asymmetric key for Formal log encryption"
  customer_master_key_spec = "RSA_4096"
  key_usage                = "ENCRYPT_DECRYPT"
}

resource "aws_kms_alias" "formal_logs" {
  name          = "alias/formal-log-encryption"
  target_key_id = aws_kms_key.formal_logs.key_id
}

data "aws_kms_public_key" "formal_logs" {
  key_id = aws_kms_key.formal_logs.arn
}

module "decryptor" {
  # Outside this repository, use
  # "git::https://github.com/formalco/decryptor.git//deploy/aws-lambda-terraform?ref=<release tag>".
  source = "../.."

  kms_key_arns            = [aws_kms_key.formal_logs.arn]
  vpc_id                  = var.vpc_id
  vpc_endpoint_subnet_ids = var.private_subnet_ids
}

resource "formal_encryption_key" "logs" {
  key_provider   = "aws-kms"
  key_id         = aws_kms_key.formal_logs.arn
  public_key_pem = data.aws_kms_public_key.formal_logs.public_key_pem
  decryptor_uri  = module.decryptor.decryptor_uri
}
