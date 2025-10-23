# S3 Bucket 9
# SOC2 Controls: CC6.1, CC6.8

resource "aws_s3_bucket" "data_9" {
  bucket = "company-data-bucket-9"

  tags = {
    Name        = "data-bucket-9"
    Environment = "prod"
    BucketID    = "9"
  }
}

resource "aws_s3_bucket_encryption" "data_9" {
  bucket = aws_s3_bucket.data_9.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.s3_9.arn
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_versioning" "data_9" {
  bucket = aws_s3_bucket.data_9.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_public_access_block" "data_9" {
  bucket = aws_s3_bucket.data_9.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_logging" "data_9" {
  bucket = aws_s3_bucket.data_9.id

  target_bucket = aws_s3_bucket.logs.id
  target_prefix = "s3-logs/data-9/"
}

resource "aws_s3_bucket_lifecycle_configuration" "data_9" {
  bucket = aws_s3_bucket.data_9.id

  rule {
    id     = "transition-to-glacier"
    status = "Enabled"

    transition {
      days          = 90
      storage_class = "GLACIER"
    }
  }
}
