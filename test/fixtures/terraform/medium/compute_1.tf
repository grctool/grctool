# EKS Compute Resources - File ${i}
# SOC2 Controls: CC6.1, CC6.6, SO2

resource "aws_eks_cluster" "cluster_${i}" {
  name     = "eks-cluster-${i}"
  role_arn = aws_iam_role.eks_cluster_${i}.arn
  version  = "1.28"

  vpc_config {
    subnet_ids              = [aws_subnet.private_1a.id, aws_subnet.private_1b.id]
    endpoint_private_access = true
    endpoint_public_access  = false
  }

  encryption_config {
    provider {
      key_arn = aws_kms_key.eks_${i}.arn
    }
    resources = ["secrets"]
  }

  enabled_cluster_log_types = ["api", "audit", "authenticator"]

  tags = {
    Name        = "eks-cluster-${i}"
    Environment = "prod"
  }
}

resource "aws_eks_node_group" "workers_${i}" {
  cluster_name    = aws_eks_cluster.cluster_${i}.name
  node_group_name = "workers-${i}"
  node_role_arn   = aws_iam_role.eks_node_${i}.arn
  subnet_ids      = [aws_subnet.private_1a.id, aws_subnet.private_1b.id]

  scaling_config {
    desired_size = 2
    max_size     = 10
    min_size     = 2
  }

  tags = {
    Name = "eks-workers-${i}"
  }
}
