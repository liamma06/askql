# AWS Deployment (Production-Grade)

## Quick AWS Setup

### 1. Create Redis on ElastiCache
```bash
aws elasticache create-cache-cluster \
    --cache-cluster-id askql-redis \
    --engine redis \
    --cache-node-type cache.t3.micro \
    --num-cache-nodes 1
```

### 2. Deploy Backend to ECS Fargate
```bash
# Build and push to ECR
aws ecr create-repository --repository-name askql-backend
docker build -t askql-backend .
docker tag askql-backend:latest 123456789.dkr.ecr.us-east-1.amazonaws.com/askql-backend:latest
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/askql-backend:latest

# Create ECS service (see previous detailed instructions)
```

### 3. Cost Estimate
- ECS Fargate: ~$15-30/month
- ElastiCache: ~$15-25/month
- ALB: ~$20/month
- Total: ~$50-75/month
