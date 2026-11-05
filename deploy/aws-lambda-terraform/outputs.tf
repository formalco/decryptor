output "decryptor_uri" {
  description = "Endpoint to set as decryptor_uri on formal_encryption_key."
  value       = "${aws_api_gateway_stage.decryptor.invoke_url}/decrypt"
}

output "rest_api_id" {
  description = "ID of the API Gateway REST API."
  value       = aws_api_gateway_rest_api.decryptor.id
}

output "vpc_endpoint_ids" {
  description = "execute-api VPC endpoints serving a PRIVATE API."
  value       = local.private ? local.vpc_endpoint_ids : []
}

output "lambda_function_name" {
  description = "Name of the Lambda function."
  value       = aws_lambda_function.decryptor.function_name
}

output "lambda_function_arn" {
  description = "ARN of the Lambda function."
  value       = aws_lambda_function.decryptor.arn
}

output "lambda_role_arn" {
  description = "ARN of the Lambda execution role."
  value       = aws_iam_role.decryptor.arn
}

output "image_uri" {
  description = "Image the Lambda function runs."
  value       = local.image_uri
}

output "cloudwatch_log_group_name" {
  description = "CloudWatch log group for the Lambda function."
  value       = aws_cloudwatch_log_group.decryptor.name
}
