data "aws_region" "current" {}

locals {
  region = data.aws_region.current.region

  # Lambda only runs single-architecture images, so each release has a tag per architecture.
  image_arch = var.architecture == "arm64" ? "arm64" : "amd64"
  image_uri  = coalesce(var.image_uri, "654654333078.dkr.ecr.${local.region}.amazonaws.com/formalco-prod-decryptor:${var.image_tag}-${local.image_arch}")

  private              = var.endpoint_type == "PRIVATE"
  create_vpc_endpoint  = local.private && length(var.vpc_endpoint_ids) == 0
  vpc_endpoint_ids     = local.create_vpc_endpoint ? aws_vpc_endpoint.execute_api[*].id : var.vpc_endpoint_ids
  lambda_in_vpc        = length(var.lambda_subnet_ids) > 0
  endpoint_ingress_vpc = local.create_vpc_endpoint && var.vpc_id != null && length(var.vpc_endpoint_ingress_cidrs) == 0

  api_policy_statements = concat(
    local.private ? [{
      Effect    = "Allow"
      Principal = "*"
      Action    = "execute-api:Invoke"
      Resource  = "execute-api:/*"
      Condition = { StringEquals = { "aws:SourceVpce" = local.vpc_endpoint_ids } }
    }] : [],
    !local.private && length(var.allowed_source_ips) > 0 ? [{
      Effect    = "Allow"
      Principal = "*"
      Action    = "execute-api:Invoke"
      Resource  = "execute-api:/*"
      Condition = { IpAddress = { "aws:SourceIp" = var.allowed_source_ips } }
    }] : [],
  )
  api_policy = length(local.api_policy_statements) > 0 ? jsonencode({
    Version   = "2012-10-17"
    Statement = local.api_policy_statements
  }) : null
}

resource "aws_iam_role" "decryptor" {
  name = "${var.name}-role"
  tags = var.tags

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
    }]
  })
}

# The key to decrypt with comes from the caller-supplied JWE, so scope the
# role to the listed keys rather than granting account-wide decrypt.
resource "aws_iam_role_policy" "kms_decrypt" {
  name = "kms-decrypt"
  role = aws_iam_role.decryptor.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["kms:Decrypt"]
      Resource = var.kms_key_arns
    }]
  })
}

resource "aws_iam_role_policy_attachment" "basic_execution" {
  role       = aws_iam_role.decryptor.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy_attachment" "vpc_execution" {
  count = local.lambda_in_vpc ? 1 : 0

  role       = aws_iam_role.decryptor.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

resource "aws_cloudwatch_log_group" "decryptor" {
  name              = "/aws/lambda/${var.name}"
  retention_in_days = var.log_retention_days
  tags              = var.tags
}

resource "aws_security_group" "lambda" {
  count = local.lambda_in_vpc ? 1 : 0

  name        = "${var.name}-lambda"
  description = "Formal decryptor Lambda function"
  vpc_id      = var.vpc_id
  tags        = merge(var.tags, { Name = "${var.name}-lambda" })

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  lifecycle {
    precondition {
      condition     = var.vpc_id != null
      error_message = "vpc_id is required when lambda_subnet_ids is set."
    }
  }
}

resource "aws_lambda_function" "decryptor" {
  function_name = var.name
  role          = aws_iam_role.decryptor.arn
  package_type  = "Image"
  image_uri     = local.image_uri
  architectures = [var.architecture]
  timeout       = 10
  tags          = var.tags

  # The function sets this origin on its POST responses; the OPTIONS preflight below uses the same value.
  environment {
    variables = {
      FORMAL_CONTROL_PLANE_URL = var.formal_control_plane_url
    }
  }

  dynamic "vpc_config" {
    for_each = local.lambda_in_vpc ? [1] : []
    content {
      subnet_ids         = var.lambda_subnet_ids
      security_group_ids = aws_security_group.lambda[*].id
    }
  }

  depends_on = [
    aws_cloudwatch_log_group.decryptor,
    aws_iam_role_policy_attachment.basic_execution,
    aws_iam_role_policy_attachment.vpc_execution,
  ]
}

data "aws_vpc" "endpoint" {
  count = local.endpoint_ingress_vpc ? 1 : 0

  id = var.vpc_id
}

resource "aws_security_group" "vpc_endpoint" {
  count = local.create_vpc_endpoint ? 1 : 0

  name        = "${var.name}-execute-api-endpoint"
  description = "execute-api VPC endpoint for the Formal decryptor"
  vpc_id      = var.vpc_id
  tags        = merge(var.tags, { Name = "${var.name}-execute-api-endpoint" })

  ingress {
    description = "HTTPS to the decryptor API"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = local.endpoint_ingress_vpc ? data.aws_vpc.endpoint[*].cidr_block : var.vpc_endpoint_ingress_cidrs
  }

  lifecycle {
    precondition {
      condition     = var.vpc_id != null && length(var.vpc_endpoint_subnet_ids) > 0
      error_message = "A PRIVATE API needs vpc_endpoint_ids, or vpc_id and vpc_endpoint_subnet_ids to create an endpoint."
    }
  }
}

resource "aws_vpc_endpoint" "execute_api" {
  count = local.create_vpc_endpoint ? 1 : 0

  vpc_id              = var.vpc_id
  service_name        = "com.amazonaws.${local.region}.execute-api"
  vpc_endpoint_type   = "Interface"
  private_dns_enabled = true
  subnet_ids          = var.vpc_endpoint_subnet_ids
  security_group_ids  = aws_security_group.vpc_endpoint[*].id
  tags                = merge(var.tags, { Name = "${var.name}-execute-api" })
}

resource "aws_api_gateway_rest_api" "decryptor" {
  name        = var.name
  description = "Formal decryptor"
  policy      = local.api_policy
  tags        = var.tags

  endpoint_configuration {
    types            = [var.endpoint_type]
    vpc_endpoint_ids = local.private ? local.vpc_endpoint_ids : null
  }

  lifecycle {
    precondition {
      condition     = local.private || length(var.vpc_endpoint_ids) == 0
      error_message = "vpc_endpoint_ids only applies to a PRIVATE API."
    }
    precondition {
      condition     = !local.private || length(var.allowed_source_ips) == 0
      error_message = "allowed_source_ips only applies to a REGIONAL API; a PRIVATE API is limited to its VPC endpoints."
    }
  }
}

resource "aws_api_gateway_resource" "decrypt" {
  rest_api_id = aws_api_gateway_rest_api.decryptor.id
  parent_id   = aws_api_gateway_rest_api.decryptor.root_resource_id
  path_part   = "decrypt"
}

resource "aws_api_gateway_method" "post" {
  rest_api_id   = aws_api_gateway_rest_api.decryptor.id
  resource_id   = aws_api_gateway_resource.decrypt.id
  http_method   = "POST"
  authorization = "NONE"
}

# A proxy integration passes the function's status, headers, and body through unchanged.
resource "aws_api_gateway_integration" "post" {
  rest_api_id             = aws_api_gateway_rest_api.decryptor.id
  resource_id             = aws_api_gateway_resource.decrypt.id
  http_method             = aws_api_gateway_method.post.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.decryptor.invoke_arn
}

resource "aws_api_gateway_method" "options" {
  rest_api_id   = aws_api_gateway_rest_api.decryptor.id
  resource_id   = aws_api_gateway_resource.decrypt.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "options" {
  rest_api_id = aws_api_gateway_rest_api.decryptor.id
  resource_id = aws_api_gateway_resource.decrypt.id
  http_method = aws_api_gateway_method.options.http_method
  type        = "MOCK"

  request_templates = {
    "application/json" = "{\"statusCode\": 200}"
  }
}

resource "aws_api_gateway_method_response" "options" {
  rest_api_id = aws_api_gateway_rest_api.decryptor.id
  resource_id = aws_api_gateway_resource.decrypt.id
  http_method = aws_api_gateway_method.options.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Origin"  = true
  }
}

resource "aws_api_gateway_integration_response" "options" {
  rest_api_id = aws_api_gateway_rest_api.decryptor.id
  resource_id = aws_api_gateway_resource.decrypt.id
  http_method = aws_api_gateway_method.options.http_method
  status_code = aws_api_gateway_method_response.options.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type'"
    "method.response.header.Access-Control-Allow-Methods" = "'OPTIONS,POST'"
    "method.response.header.Access-Control-Allow-Origin"  = "'${var.formal_control_plane_url}'"
  }

  depends_on = [aws_api_gateway_integration.options]
}

resource "aws_api_gateway_deployment" "decryptor" {
  rest_api_id = aws_api_gateway_rest_api.decryptor.id

  # Policy and CORS header changes update resources in place (same IDs) and
  # only take effect once the API is redeployed.
  triggers = {
    redeployment = sha1(jsonencode([
      aws_api_gateway_rest_api.decryptor.policy,
      aws_api_gateway_integration_response.options.response_parameters,
      aws_api_gateway_resource.decrypt.id,
      aws_api_gateway_method.post.id,
      aws_api_gateway_integration.post.id,
      aws_api_gateway_method.options.id,
      aws_api_gateway_integration.options.id,
      aws_api_gateway_method_response.options.id,
      aws_api_gateway_integration_response.options.id,
    ]))
  }

  lifecycle {
    create_before_destroy = true
  }

  depends_on = [
    aws_api_gateway_integration.post,
    aws_api_gateway_integration_response.options,
  ]
}

resource "aws_api_gateway_stage" "decryptor" {
  rest_api_id   = aws_api_gateway_rest_api.decryptor.id
  deployment_id = aws_api_gateway_deployment.decryptor.id
  stage_name    = var.stage_name
  tags          = var.tags
}

resource "aws_lambda_permission" "api_gateway" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.decryptor.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.decryptor.execution_arn}/*/*"
}
