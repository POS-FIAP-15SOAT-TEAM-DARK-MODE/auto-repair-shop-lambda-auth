output "environment" {
  value = local.environment
}

output "function_name" {
  value = aws_lambda_function.customer_login.function_name
}

output "function_arn" {
  description = "Consumed by infra-k8s's API Gateway state (issue #5) for the Lambda proxy integration."
  value       = aws_lambda_function.customer_login.arn
}

output "invoke_url" {
  description = "Direct HTTPS URL for testing ahead of the API Gateway integration. POST { \"cpf\": \"...\" } here."
  value       = aws_lambda_function_url.customer_login.function_url
}

output "security_group_id" {
  description = "Consumed by auto-repair-shop-infra-db to add an RDS ingress rule allowing this lambda in on 5432."
  value       = aws_security_group.lambda.id
}
