# KMS Keys Set 1
# SOC2 Controls: CC6.1, CC6.8

resource "aws_kms_key" "eks_1" {
  description             = "EKS encryption key 1"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "eks-key-1"
  }
}

resource "aws_kms_alias" "eks_1" {
  name          = "alias/eks-1"
  target_key_id = aws_kms_key.eks_1.key_id
}

resource "aws_kms_key" "rds_1" {
  description             = "RDS encryption key 1"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "rds-key-1"
  }
}

resource "aws_kms_alias" "rds_1" {
  name          = "alias/rds-1"
  target_key_id = aws_kms_key.rds_1.key_id
}

resource "aws_kms_key" "s3_1" {
  description             = "S3 encryption key 1"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "s3-key-1"
  }
}

resource "aws_kms_alias" "s3_1" {
  name          = "alias/s3-1"
  target_key_id = aws_kms_key.s3_1.key_id
}
