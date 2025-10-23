# KMS Keys Set 5
# SOC2 Controls: CC6.1, CC6.8

resource "aws_kms_key" "eks_5" {
  description             = "EKS encryption key 5"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "eks-key-5"
  }
}

resource "aws_kms_alias" "eks_5" {
  name          = "alias/eks-5"
  target_key_id = aws_kms_key.eks_5.key_id
}

resource "aws_kms_key" "rds_5" {
  description             = "RDS encryption key 5"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "rds-key-5"
  }
}

resource "aws_kms_alias" "rds_5" {
  name          = "alias/rds-5"
  target_key_id = aws_kms_key.rds_5.key_id
}

resource "aws_kms_key" "s3_5" {
  description             = "S3 encryption key 5"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "s3-key-5"
  }
}

resource "aws_kms_alias" "s3_5" {
  name          = "alias/s3-5"
  target_key_id = aws_kms_key.s3_5.key_id
}
