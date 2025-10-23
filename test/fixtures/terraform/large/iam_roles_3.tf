# IAM Roles Set 3
# SOC2 Controls: CC6.1, CC6.2

resource "aws_iam_role" "eks_cluster_3" {
  name = "eks-cluster-role-3"

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

  tags = {
    Name = "eks-cluster-role-3"
  }
}

resource "aws_iam_role_policy_attachment" "eks_cluster_3_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
  role       = aws_iam_role.eks_cluster_3.name
}

resource "aws_iam_role" "eks_node_3" {
  name = "eks-node-role-3"

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

  tags = {
    Name = "eks-node-role-3"
  }
}

resource "aws_iam_role_policy_attachment" "eks_node_3_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
  role       = aws_iam_role.eks_node_3.name
}

resource "aws_iam_role" "app_3" {
  name = "app-role-3"

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

  tags = {
    Name = "app-role-3"
  }
}
