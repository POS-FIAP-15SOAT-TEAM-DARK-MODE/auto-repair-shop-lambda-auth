terraform {
  required_version = ">= 1.10"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~> 2.5"
    }
  }

  # bucket + key passed via -backend-config by the workflow; same bucket as
  # infra-k8s/infra-db, different key so states never collide.
  backend "s3" {
    key    = "lambda-auth/terraform.tfstate"
    region = "us-east-1"
  }
}

provider "aws" {
  region = var.region
}
