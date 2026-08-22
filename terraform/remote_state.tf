# Network (VPC + private subnets) comes from the sibling
# auto-repair-shop-infra-k8s repo's `aws` state — same S3 bucket, different
# key. The DB host + app secret ARN come from auto-repair-shop-infra-db's
# state. This repo never creates VPC or RDS resources, only reads them.
data "terraform_remote_state" "network" {
  backend   = "s3"
  workspace = terraform.workspace

  config = {
    bucket = var.state_bucket
    key    = "aws/terraform.tfstate"
    region = var.region
  }
}

data "terraform_remote_state" "db" {
  backend   = "s3"
  workspace = terraform.workspace

  config = {
    bucket = var.state_bucket
    key    = "db/terraform.tfstate"
    region = var.region
  }
}
