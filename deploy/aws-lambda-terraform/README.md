# AWS Lambda via Terraform

A Terraform module that runs the decryptor as a Lambda function behind API Gateway. Use its `decryptor_uri` output as the `decryptor_uri` on a `formal_encryption_key`.

[`examples/aws-kms`](examples/aws-kms) creates an asymmetric KMS key, deploys this module, and registers the key with Formal.

## Usage

```hcl
module "decryptor" {
  source = "git::https://github.com/formalco/decryptor.git//deploy/aws-lambda-terraform?ref=<release tag>"

  kms_key_arns            = [aws_kms_key.formal_logs.arn]
  vpc_id                  = "vpc-xxxxxxxxx"
  vpc_endpoint_subnet_ids = ["subnet-xxxxxxxxx", "subnet-yyyyyyyyy"]
}

resource "formal_encryption_key" "logs" {
  key_provider   = "aws-kms"
  key_id         = aws_kms_key.formal_logs.arn
  public_key_pem = data.aws_kms_public_key.formal_logs.public_key_pem
  decryptor_uri  = module.decryptor.decryptor_uri
}
```

## Image

By default the function runs Formal's published image:

```
654654333078.dkr.ecr.<region>.amazonaws.com/formalco-prod-decryptor:<image_tag>-<arm64|amd64>
```

The image is replicated to `eu-west-1`, `us-east-1`, `us-east-2`, `us-west-1`, `us-west-2`, and `ap-southeast-1`. The module uses the replica in the provider's region.

Pulling it requires your AWS account or organization to be allowlisted by Formal. The identity running Terraform also needs `ecr:BatchGetImage` and `ecr:GetDownloadUrlForLayer` on the repository.

To run your own build, build the repository-root `Dockerfile` for one architecture with `--provenance=false`. Push it to ECR and set `image_uri`. Lambda only accepts single-architecture images.

## Network access

There is no authentication between the browser and the decryptor, so restrict who can reach it.

- `endpoint_type = "PRIVATE"` (default) serves the API only through execute-api VPC endpoints. Users reach it over a VPN. The module creates an endpoint in `vpc_id`, or reuses `vpc_endpoint_ids`. A VPC can have only one execute-api endpoint with private DNS, so reuse it if your VPC already has one.
- `endpoint_type = "REGIONAL"` serves the API publicly. Set `allowed_source_ips` to your VPN or office egress IPs to limit callers.

The function itself runs outside your VPC unless you set `lambda_subnet_ids`. Attached subnets need a route to KMS, through a NAT gateway or a KMS VPC endpoint.

## Self-hosted console

The browser only accepts decryptor responses for the Formal Console origin in `formal_control_plane_url`. It defaults to `https://app.formal.ai`. If you run a self-hosted control plane, set it to your console's origin, e.g. `formal_control_plane_url = "https://formal.example.com"`.

## Key rotation

Each encrypted field records the KMS key that sealed it. When you replace a key, add the new ARN to `kms_key_arns` and keep the old one until its fields no longer need decrypting.

## Inputs

| Name | Description | Default |
|------|-------------|---------|
| `kms_key_arns` | KMS keys the decryptor may decrypt with | required |
| `name` | Function name; prefixes the role, security groups, and API | `formal-decryptor` |
| `image_tag` | Decryptor release to run | `v0.1.1` |
| `image_uri` | Image to run instead of Formal's ECR image | `null` |
| `architecture` | `arm64` or `x86_64` | `arm64` |
| `endpoint_type` | `PRIVATE` or `REGIONAL` | `PRIVATE` |
| `vpc_endpoint_ids` | Existing execute-api endpoints for a `PRIVATE` API | `[]` |
| `vpc_id` | VPC for the created endpoint and for `lambda_subnet_ids` | `null` |
| `vpc_endpoint_subnet_ids` | Subnets for the created endpoint | `[]` |
| `vpc_endpoint_ingress_cidrs` | CIDRs allowed to reach the created endpoint | VPC CIDR |
| `allowed_source_ips` | Callers allowed to reach a `REGIONAL` API | any |
| `lambda_subnet_ids` | Subnets to attach the function to | none |
| `formal_control_plane_url` | Formal Console origin allowed to call the decryptor | `https://app.formal.ai` |
| `stage_name` | API Gateway stage name | `prod` |
| `log_retention_days` | CloudWatch log retention | `14` |
| `tags` | Tags for every resource | `{}` |

## Outputs

| Name | Description |
|------|-------------|
| `decryptor_uri` | Endpoint to set as `decryptor_uri` |
| `rest_api_id` | API Gateway REST API ID |
| `vpc_endpoint_ids` | execute-api endpoints serving a `PRIVATE` API |
| `lambda_function_name` | Lambda function name |
| `lambda_function_arn` | Lambda function ARN |
| `lambda_role_arn` | Lambda execution role ARN |
| `image_uri` | Image the function runs |
| `cloudwatch_log_group_name` | Lambda log group |
