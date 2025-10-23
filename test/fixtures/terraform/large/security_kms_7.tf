# KMS Keys Set 7
# SOC2 Controls: CC6.1, CC6.8

resource "aws_kms_key" "eks_7" {
  description             = "EKS encryption key 7"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "eks-key-7"
  }
}

resource "aws_kms_alias" "eks_7" {
  name          = "alias/eks-7"
  target_key_id = aws_kms_key.eks_7.key_id
}

resource "aws_kms_key" "rds_7" {
  description             = "RDS encryption key 7"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "rds-key-7"
  }
}

resource "aws_kms_alias" "rds_7" {
  name          = "alias/rds-7"
  target_key_id = aws_kms_key.rds_7.key_id
}

resource "aws_kms_key" "s3_7" {
  description             = "S3 encryption key 7"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "s3-key-7"
  }
}

resource "aws_kms_alias" "s3_7" {
  name          = "alias/s3-7"
  target_key_id = aws_kms_key.s3_7.key_id
}
