# Storage Resources - File ${i}
# SOC2 Controls: CC6.1, CC6.8

resource "aws_s3_bucket" "data_${i}" {
  bucket = "company-data-${i}"

  tags = {
    Name        = "data-bucket-${i}"
    Environment = "prod"
  }
}

resource "aws_s3_bucket_encryption" "data_${i}" {
  bucket = aws_s3_bucket.data_${i}.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.s3_${i}.arn
    }
  }
}

resource "aws_s3_bucket_versioning" "data_${i}" {
  bucket = aws_s3_bucket.data_${i}.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_public_access_block" "data_${i}" {
  bucket = aws_s3_bucket.data_${i}.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_logging" "data_${i}" {
  bucket = aws_s3_bucket.data_${i}.id

  target_bucket = aws_s3_bucket.logs.id
  target_prefix = "s3-access-logs/data-${i}/"
}
