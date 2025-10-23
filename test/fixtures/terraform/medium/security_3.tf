# Security Resources - File ${i}
# SOC2 Controls: CC6.1, CC6.8

resource "aws_kms_key" "eks_${i}" {
  description             = "EKS encryption key ${i}"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Name = "eks-key-${i}"
  }
}

resource "aws_kms_key" "rds_${i}" {
  description             = "RDS encryption key ${i}"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Name = "rds-key-${i}"
  }
}

resource "aws_kms_key" "s3_${i}" {
  description             = "S3 encryption key ${i}"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Name = "s3-key-${i}"
  }
}

resource "aws_iam_role" "eks_cluster_${i}" {
  name = "eks-cluster-role-${i}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "eks.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_role" "eks_node_${i}" {
  name = "eks-node-role-${i}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
    }]
  })
}
