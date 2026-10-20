output "cluster_name" {
  description = "EKS cluster name"
  value       = aws_eks_cluster.skypod.name
}

output "cluster_endpoint" {
  description = "Kubernetes API endpoint"
  value       = aws_eks_cluster.skypod.endpoint
}

output "region" {
  description = "AWS region the cluster runs in"
  value       = var.region
}

output "kubeconfig_command" {
  description = "Run this to point kubectl at the cluster"
  value       = "aws eks update-kubeconfig --name ${aws_eks_cluster.skypod.name} --region ${var.region}"
}
