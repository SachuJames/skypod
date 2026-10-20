variable "region" {
  description = "AWS region for the cluster"
  type        = string
  default     = "ap-south-1"
}

variable "cluster_name" {
  description = "EKS cluster name"
  type        = string
  default     = "skypod"
}

variable "kubernetes_version" {
  description = "EKS control plane version"
  type        = string
  default     = "1.30"
}

variable "node_instance_types" {
  description = "Instance types for the managed node group"
  type        = list(string)
  default     = ["t3.medium"]
}

variable "node_desired" {
  description = "Desired node count"
  type        = number
  default     = 2
}

variable "node_min" {
  description = "Minimum node count"
  type        = number
  default     = 2
}

variable "node_max" {
  description = "Maximum node count"
  type        = number
  default     = 4
}

variable "vpc_cidr" {
  description = "CIDR for the cluster VPC"
  type        = string
  default     = "10.0.0.0/16"
}
