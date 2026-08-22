variable "region" {
  type    = string
  default = "us-east-1"
}

variable "project" {
  type    = string
  default = "auto-repair-shop"
}

variable "state_bucket" {
  description = "S3 bucket holding the remote state — the same bucket created by auto-repair-shop-infra-k8s's bootstrap state. Passed by the workflow, derived from the AWS account id."
  type        = string
}

variable "manage_iam" {
  description = "Let Terraform create the lambda's own execution role. Set to false on restricted accounts (Learner Lab): the lambda then reuses execution_role_arn (e.g. LabRole) instead."
  type        = bool
  default     = true
}

variable "execution_role_arn" {
  description = "Pre-existing IAM role used as the lambda's execution role when manage_iam = false (e.g. the Learner Lab LabRole)."
  type        = string
  default     = ""
}

variable "lambda_zip_path" {
  description = "Path to the built deployment package (bootstrap binary zipped), produced by the CI workflow before terraform apply."
  type        = string
  default     = "../build/lambda.zip"
}

variable "jwt_expires_in" {
  description = "Token lifetime, same convention as the app's JWT_EXPIRES_IN."
  type        = string
  default     = "24h"
}
