variable "aws_region" {
  description = "AWS region to deploy into."
  type        = string
  default     = "us-east-1"
}

variable "vpc_id" {
  description = "VPC your users reach over a VPN."
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnets in vpc_id for the execute-api VPC endpoint. Use two or more availability zones."
  type        = list(string)
}
