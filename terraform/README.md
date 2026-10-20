# Terraform: the AWS path

This directory stands up the production home for skypod on AWS: a VPC
with public and private subnets across two AZs, an EKS cluster, and a
managed node group in the private subnets. No external modules, so the
whole thing is readable in three files.

## Cost, stated up front

This costs real money. The EKS control plane bills per hour and two
t3.medium nodes bill per hour on top of the NAT gateway and the EBS
volumes they use. Roughly: tens of dollars a month just to leave it
running, more if you scale up. Create it for a demo, show what you need
to show, then destroy it.

## Run it

```
cd terraform
terraform init
terraform plan -out plan.out
terraform apply plan.out
```

Connect kubectl afterwards with the command from the outputs, then
install the chart:

```
eval "$(terraform output -raw kubeconfig_command)"
helm upgrade --install skypod ../charts/skypod --namespace skypod --create-namespace
```

## Tear it down

```
terraform destroy
```
