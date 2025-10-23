# KMS Keys Set 3
# SOC2 Controls: CC6.1, CC6.8

resource "aws_kms_key" "eks_3" {
  description             = "EKS encryption key 3"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "eks-key-3"
  }
}

resource "aws_kms_alias" "eks_3" {
  name          = "alias/eks-3"
  target_key_id = aws_kms_key.eks_3.key_id
}

resource "aws_kms_key" "rds_3" {
  description             = "RDS encryption key 3"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "rds-key-3"
  }
}

resource "aws_kms_alias" "rds_3" {
  name          = "alias/rds-3"
  target_key_id = aws_kms_key.rds_3.key_id
}

resource "aws_kms_key" "s3_3" {
  description             = "S3 encryption key 3"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "s3-key-3"
  }
}

resource "aws_kms_alias" "s3_3" {
  name          = "alias/s3-3"
  target_key_id = aws_kms_key.s3_3.key_id
}
