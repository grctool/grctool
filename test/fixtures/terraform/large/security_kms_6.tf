# KMS Keys Set 6
# SOC2 Controls: CC6.1, CC6.8

resource "aws_kms_key" "eks_6" {
  description             = "EKS encryption key 6"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "eks-key-6"
  }
}

resource "aws_kms_alias" "eks_6" {
  name          = "alias/eks-6"
  target_key_id = aws_kms_key.eks_6.key_id
}

resource "aws_kms_key" "rds_6" {
  description             = "RDS encryption key 6"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "rds-key-6"
  }
}

resource "aws_kms_alias" "rds_6" {
  name          = "alias/rds-6"
  target_key_id = aws_kms_key.rds_6.key_id
}

resource "aws_kms_key" "s3_6" {
  description             = "S3 encryption key 6"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "s3-key-6"
  }
}

resource "aws_kms_alias" "s3_6" {
  name          = "alias/s3-6"
  target_key_id = aws_kms_key.s3_6.key_id
}
