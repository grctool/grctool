# Basic S3 bucket configuration for minimal testing
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Simple S3 bucket
resource "aws_s3_bucket" "basic_bucket" {
  bucket = "basic-test-bucket-${random_id.bucket_suffix.hex}"

  tags = {
    Environment = "test"
    Purpose     = "basic-testing"
  }
}

# Basic server-side encryption
resource "aws_s3_bucket_server_side_encryption_configuration" "basic_bucket" {
  bucket = aws_s3_bucket.basic_bucket.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# Public access blocking
resource "aws_s3_bucket_public_access_block" "basic_bucket" {
  bucket = aws_s3_bucket.basic_bucket.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "random_id" "bucket_suffix" {
  byte_length = 4
}