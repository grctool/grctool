# KMS Keys Set 4
# SOC2 Controls: CC6.1, CC6.8

resource "aws_kms_key" "eks_4" {
  description             = "EKS encryption key 4"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "eks-key-4"
  }
}

resource "aws_kms_alias" "eks_4" {
  name          = "alias/eks-4"
  target_key_id = aws_kms_key.eks_4.key_id
}

resource "aws_kms_key" "rds_4" {
  description             = "RDS encryption key 4"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "rds-key-4"
  }
}

resource "aws_kms_alias" "rds_4" {
  name          = "alias/rds-4"
  target_key_id = aws_kms_key.rds_4.key_id
}

resource "aws_kms_key" "s3_4" {
  description             = "S3 encryption key 4"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "s3-key-4"
  }
}

resource "aws_kms_alias" "s3_4" {
  name          = "alias/s3-4"
  target_key_id = aws_kms_key.s3_4.key_id
}
