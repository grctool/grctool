# EKS Cluster 7
# SOC2 Controls: CC6.1, CC6.6, SO2

resource "aws_eks_cluster" "cluster_7" {
  name     = "eks-cluster-7"
  role_arn = aws_iam_role.eks_cluster_7.arn
  version  = "1.28"

  vpc_config {
    subnet_ids              = [aws_subnet.region_1_private_a.id, aws_subnet.region_1_private_b.id]
    endpoint_private_access = true
    endpoint_public_access  = false
  }

  encryption_config {
    provider {
      key_arn = aws_kms_key.eks_7.arn
    }
    resources = ["secrets"]
  }

  enabled_cluster_log_types = ["api", "audit", "authenticator", "controllerManager", "scheduler"]

  tags = {
    Name        = "eks-cluster-7"
    Environment = "prod"
    ClusterID   = "7"
  }
}

resource "aws_eks_node_group" "workers_7_a" {
  cluster_name    = aws_eks_cluster.cluster_7.name
  node_group_name = "workers-7-a"
  node_role_arn   = aws_iam_role.eks_node_7.arn
  subnet_ids      = [aws_subnet.region_1_private_a.id]

  scaling_config {
    desired_size = 3
    max_size     = 20
    min_size     = 3
  }

  tags = {
    Name = "eks-workers-7-a"
  }
}

resource "aws_eks_node_group" "workers_7_b" {
  cluster_name    = aws_eks_cluster.cluster_7.name
  node_group_name = "workers-7-b"
  node_role_arn   = aws_iam_role.eks_node_7.arn
  subnet_ids      = [aws_subnet.region_1_private_b.id]

  scaling_config {
    desired_size = 3
    max_size     = 20
    min_size     = 3
  }

  tags = {
    Name = "eks-workers-7-b"
  }
}
