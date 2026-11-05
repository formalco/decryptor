variable "name" {
  description = "Name of the Lambda function. Also prefixes the IAM role, security groups, and API Gateway."
  type        = string
  default     = "formal-decryptor"
}

variable "kms_key_arns" {
  description = "ARNs of the asymmetric KMS keys the decryptor may decrypt with. Keep retired keys listed until fields encrypted with them no longer need decrypting."
  type        = list(string)

  validation {
    condition     = length(var.kms_key_arns) > 0
    error_message = "kms_key_arns must contain at least one KMS key ARN."
  }
}

variable "image_tag" {
  description = "Decryptor release to run, e.g. v0.1.1. Ignored when image_uri is set."
  type        = string
  default     = "v0.1.1"
}

variable "image_uri" {
  description = "Full image URI to run instead of Formal's ECR image, e.g. a copy in your own ECR. Must be a single-architecture image matching architecture."
  type        = string
  default     = null
}

variable "architecture" {
  description = "Lambda instruction set architecture: arm64 or x86_64."
  type        = string
  default     = "arm64"

  validation {
    condition     = contains(["arm64", "x86_64"], var.architecture)
    error_message = "architecture must be arm64 or x86_64."
  }
}

variable "endpoint_type" {
  description = "PRIVATE serves the API only through execute-api VPC endpoints; users reach it over a VPN. REGIONAL serves it publicly, optionally limited to allowed_source_ips."
  type        = string
  default     = "PRIVATE"

  validation {
    condition     = contains(["PRIVATE", "REGIONAL"], var.endpoint_type)
    error_message = "endpoint_type must be PRIVATE or REGIONAL."
  }
}

variable "vpc_endpoint_ids" {
  description = "Existing execute-api VPC endpoints to serve a PRIVATE API through. Leave empty to create one in vpc_id. A VPC can have only one execute-api endpoint with private DNS, so reuse it if the VPC already has one."
  type        = list(string)
  default     = []
}

variable "vpc_id" {
  description = "VPC to create the execute-api VPC endpoint in. Required for a PRIVATE API when vpc_endpoint_ids is empty."
  type        = string
  default     = null
}

variable "vpc_endpoint_subnet_ids" {
  description = "Subnets for the created execute-api VPC endpoint. Use two or more availability zones."
  type        = list(string)
  default     = []
}

variable "vpc_endpoint_ingress_cidrs" {
  description = "CIDRs allowed to reach the created VPC endpoint on 443. Defaults to the VPC CIDR. Add your VPN client CIDR if VPN traffic is not translated into the VPC range."
  type        = list(string)
  default     = []
}

variable "allowed_source_ips" {
  description = "For a REGIONAL API, the public IPs or CIDRs allowed to call it, e.g. VPN egress IPs. Empty allows any caller."
  type        = list(string)
  default     = []
}

variable "lambda_subnet_ids" {
  description = "Subnets to attach the Lambda function to. Leave empty to run it outside a VPC. Attached subnets need a route to KMS through a NAT gateway or a KMS VPC endpoint."
  type        = list(string)
  default     = []
}

variable "stage_name" {
  description = "API Gateway stage name."
  type        = string
  default     = "prod"
}

variable "log_retention_days" {
  description = "CloudWatch log retention in days."
  type        = number
  default     = 14
}

variable "tags" {
  description = "Tags to apply to every resource."
  type        = map(string)
  default     = {}
}
