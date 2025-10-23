# KMS Keys Set 8
# SOC2 Controls: CC6.1, CC6.8

resource "aws_kms_key" "eks_8" {
  description             = "EKS encryption key 8"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "eks-key-8"
  }
}

resource "aws_kms_alias" "eks_8" {
  name          = "alias/eks-8"
  target_key_id = aws_kms_key.eks_8.key_id
}

resource "aws_kms_key" "rds_8" {
  description             = "RDS encryption key 8"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "rds-key-8"
  }
}

resource "aws_kms_alias" "rds_8" {
  name          = "alias/rds-8"
  target_key_id = aws_kms_key.rds_8.key_id
}

resource "aws_kms_key" "s3_8" {
  description             = "S3 encryption key 8"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "s3-key-8"
  }
}

resource "aws_kms_alias" "s3_8" {
  name          = "alias/s3-8"
  target_key_id = aws_kms_key.s3_8.key_id
}
