# KMS Keys Set 2
# SOC2 Controls: CC6.1, CC6.8

resource "aws_kms_key" "eks_2" {
  description             = "EKS encryption key 2"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "eks-key-2"
  }
}

resource "aws_kms_alias" "eks_2" {
  name          = "alias/eks-2"
  target_key_id = aws_kms_key.eks_2.key_id
}

resource "aws_kms_key" "rds_2" {
  description             = "RDS encryption key 2"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "rds-key-2"
  }
}

resource "aws_kms_alias" "rds_2" {
  name          = "alias/rds-2"
  target_key_id = aws_kms_key.rds_2.key_id
}

resource "aws_kms_key" "s3_2" {
  description             = "S3 encryption key 2"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = {
    Name = "s3-key-2"
  }
}

resource "aws_kms_alias" "s3_2" {
  name          = "alias/s3-2"
  target_key_id = aws_kms_key.s3_2.key_id
}
