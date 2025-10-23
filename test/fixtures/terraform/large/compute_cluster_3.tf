# EKS Cluster 3
# SOC2 Controls: CC6.1, CC6.6, SO2

resource "aws_eks_cluster" "cluster_3" {
  name     = "eks-cluster-3"
  role_arn = aws_iam_role.eks_cluster_3.arn
  version  = "1.28"

  vpc_config {
    subnet_ids              = [aws_subnet.region_1_private_a.id, aws_subnet.region_1_private_b.id]
    endpoint_private_access = true
    endpoint_public_access  = false
  }

  encryption_config {
    provider {
      key_arn = aws_kms_key.eks_3.arn
    }
    resources = ["secrets"]
  }

  enabled_cluster_log_types = ["api", "audit", "authenticator", "controllerManager", "scheduler"]

  tags = {
    Name        = "eks-cluster-3"
    Environment = "prod"
    ClusterID   = "3"
  }
}

resource "aws_eks_node_group" "workers_3_a" {
  cluster_name    = aws_eks_cluster.cluster_3.name
  node_group_name = "workers-3-a"
  node_role_arn   = aws_iam_role.eks_node_3.arn
  subnet_ids      = [aws_subnet.region_1_private_a.id]

  scaling_config {
    desired_size = 3
    max_size     = 20
    min_size     = 3
  }

  tags = {
    Name = "eks-workers-3-a"
  }
}

resource "aws_eks_node_group" "workers_3_b" {
  cluster_name    = aws_eks_cluster.cluster_3.name
  node_group_name = "workers-3-b"
  node_role_arn   = aws_iam_role.eks_node_3.arn
  subnet_ids      = [aws_subnet.region_1_private_b.id]

  scaling_config {
    desired_size = 3
    max_size     = 20
    min_size     = 3
  }

  tags = {
    Name = "eks-workers-3-b"
  }
}
