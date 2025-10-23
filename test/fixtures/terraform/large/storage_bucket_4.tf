# S3 Bucket 4
# SOC2 Controls: CC6.1, CC6.8

resource "aws_s3_bucket" "data_4" {
  bucket = "company-data-bucket-4"

  tags = {
    Name        = "data-bucket-4"
    Environment = "prod"
    BucketID    = "4"
  }
}

resource "aws_s3_bucket_encryption" "data_4" {
  bucket = aws_s3_bucket.data_4.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.s3_4.arn
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_versioning" "data_4" {
  bucket = aws_s3_bucket.data_4.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_public_access_block" "data_4" {
  bucket = aws_s3_bucket.data_4.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_logging" "data_4" {
  bucket = aws_s3_bucket.data_4.id

  target_bucket = aws_s3_bucket.logs.id
  target_prefix = "s3-logs/data-4/"
}

resource "aws_s3_bucket_lifecycle_configuration" "data_4" {
  bucket = aws_s3_bucket.data_4.id

  rule {
    id     = "transition-to-glacier"
    status = "Enabled"

    transition {
      days          = 90
      storage_class = "GLACIER"
    }
  }
}
