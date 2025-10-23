# EKS Cluster 1
# SOC2 Controls: CC6.1, CC6.6, SO2

resource "aws_eks_cluster" "cluster_1" {
  name     = "eks-cluster-1"
  role_arn = aws_iam_role.eks_cluster_1.arn
  version  = "1.28"

  vpc_config {
    subnet_ids              = [aws_subnet.region_1_private_a.id, aws_subnet.region_1_private_b.id]
    endpoint_private_access = true
    endpoint_public_access  = false
  }

  encryption_config {
    provider {
      key_arn = aws_kms_key.eks_1.arn
    }
    resources = ["secrets"]
  }

  enabled_cluster_log_types = ["api", "audit", "authenticator", "controllerManager", "scheduler"]

  tags = {
    Name        = "eks-cluster-1"
    Environment = "prod"
    ClusterID   = "1"
  }
}

resource "aws_eks_node_group" "workers_1_a" {
  cluster_name    = aws_eks_cluster.cluster_1.name
  node_group_name = "workers-1-a"
  node_role_arn   = aws_iam_role.eks_node_1.arn
  subnet_ids      = [aws_subnet.region_1_private_a.id]

  scaling_config {
    desired_size = 3
    max_size     = 20
    min_size     = 3
  }

  tags = {
    Name = "eks-workers-1-a"
  }
}

resource "aws_eks_node_group" "workers_1_b" {
  cluster_name    = aws_eks_cluster.cluster_1.name
  node_group_name = "workers-1-b"
  node_role_arn   = aws_iam_role.eks_node_1.arn
  subnet_ids      = [aws_subnet.region_1_private_b.id]

  scaling_config {
    desired_size = 3
    max_size     = 20
    min_size     = 3
  }

  tags = {
    Name = "eks-workers-1-b"
  }
}
