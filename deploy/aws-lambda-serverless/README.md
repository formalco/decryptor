# AWS Lambda via Serverless

Deploys the decryptor as a Lambda behind a private REST API Gateway, reachable only through your `execute-api` VPC endpoint.

## Prerequisites

- A Serverless license
- The Serverless CLI (`npm i -E serverless@4.21.1 -g`)
- AWS credentials able to deploy API Gateways, Lambdas, and VPC networking
- An existing VPC with two private subnets, a security group for the Lambda, and an `execute-api` interface VPC endpoint (private DNS enabled)
- These environment variables:
  - `KMS_KEY_ARN` — the decryption key ARN; the Lambda role is scoped to `kms:Decrypt` on this key only
  - `VPC_ENDPOINT_ID` — the `execute-api` VPC endpoint the API is locked to
  - `SECURITY_GROUP_ID` — security group for the Lambda
  - `SUBNET_ID_1`, `SUBNET_ID_2` — private subnets for the Lambda
  - `FORMAL_CONTROL_PLANE_URL` (optional) — the Formal Console origin allowed to call the decryptor, defaulting to `https://app.formal.ai`; set it to your console's origin, e.g. `https://formal.example.com`, for a self-hosted control plane

## Deployment steps

```bash
npm i -E serverless@4.21.1 -g
export KMS_KEY_ARN=arn:aws:kms:<region>:<account>:key/<id>
export VPC_ENDPOINT_ID=vpce-...
export SECURITY_GROUP_ID=sg-...
export SUBNET_ID_1=subnet-... SUBNET_ID_2=subnet-...
# build the Lambda binary into this directory (Serverless packages ./bootstrap)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o deploy/aws-lambda-serverless/bootstrap .
cd deploy/aws-lambda-serverless && sls deploy
```

Note: you will need a license from Serverless to use the CLI. This creates a CloudFormation stack in your AWS environment.

To expose the API publicly instead, drop `endpointType`, `resourcePolicy`, and `vpc` from `serverless.yml`.
