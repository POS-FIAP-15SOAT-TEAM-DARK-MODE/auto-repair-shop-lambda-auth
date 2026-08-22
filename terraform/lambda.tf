locals {
  environment = terraform.workspace == "default" ? "stg" : terraform.workspace
  name        = "${var.project}-${local.environment}-lambda-auth"

  tags = {
    Project     = var.project
    Environment = local.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/${local.name}"
  retention_in_days = 14
  tags              = local.tags
}

# Everything in this file is IAM and is skipped when manage_iam = false
# (restricted accounts such as AWS Academy Learner Lab). In that mode the
# lambda reuses execution_role_arn (e.g. LabRole) instead.
data "aws_iam_policy_document" "lambda_assume" {
  count = var.manage_iam ? 1 : 0

  statement {
    actions = ["sts:AssumeRoleWithWebIdentity", "sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "lambda" {
  count              = var.manage_iam ? 1 : 0
  name               = "${local.name}-role"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume[0].json
  tags               = local.tags
}

resource "aws_iam_role_policy_attachment" "vpc_access" {
  count      = var.manage_iam ? 1 : 0
  role       = aws_iam_role.lambda[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

data "aws_iam_policy_document" "lambda_secrets" {
  count = var.manage_iam ? 1 : 0

  statement {
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [data.terraform_remote_state.db.outputs.app_secret_arn]
  }
}

resource "aws_iam_role_policy" "lambda_secrets" {
  count  = var.manage_iam ? 1 : 0
  name   = "read-app-secret"
  role   = aws_iam_role.lambda[0].id
  policy = data.aws_iam_policy_document.lambda_secrets[0].json
}

locals {
  execution_role_arn = var.manage_iam ? aws_iam_role.lambda[0].arn : var.execution_role_arn
}

resource "aws_lambda_function" "customer_login" {
  function_name = local.name
  role          = local.execution_role_arn

  runtime       = "provided.al2023"
  architectures = ["arm64"]
  handler       = "bootstrap"

  filename         = var.lambda_zip_path
  source_code_hash = filebase64sha256(var.lambda_zip_path)

  timeout     = 10
  memory_size = 128

  vpc_config {
    subnet_ids = data.terraform_remote_state.network.outputs.private_subnets
    # Owned by auto-repair-shop-infra-db, not created here, to avoid a
    # circular cross-repo dependency (see that repo's lambda_access.tf).
    security_group_ids = [data.terraform_remote_state.db.outputs.lambda_access_security_group_id]
  }

  environment {
    variables = {
      DB_HOST        = data.terraform_remote_state.db.outputs.db_host
      DB_PORT        = tostring(data.terraform_remote_state.db.outputs.db_port)
      DB_USER        = "postgres"
      DB_NAME        = "autorepairshop"
      APP_SECRET_ID  = data.terraform_remote_state.db.outputs.app_secret_arn
      JWT_EXPIRES_IN = var.jwt_expires_in
    }
  }

  depends_on = [aws_cloudwatch_log_group.lambda]
  tags       = local.tags
}

# Public HTTPS invoke URL for direct testing while infra-k8s issue #5 (API
# Gateway) is still in progress. auth_type NONE is intentional: this endpoint
# is meant to be publicly reachable regardless of gateway — it's how a
# customer gets a token in the first place, same as the app's own
# POST /v1/auth/login.
resource "aws_lambda_function_url" "customer_login" {
  function_name      = aws_lambda_function.customer_login.function_name
  authorization_type = "NONE"
}
