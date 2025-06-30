# Deploy to Google Cloud Platform

## Prerequisites
1. Install Google Cloud CLI: https://cloud.google.com/sdk/docs/install
2. Create GCP project: https://console.cloud.google.com

## Step 1: Setup and Authentication

### Option A: Using Terminal (gcloud CLI)
```bash
# Login to GCP
gcloud auth login

# Set your project
gcloud config set project YOUR_PROJECT_ID

# Enable required APIs
gcloud services enable run.googleapis.com
gcloud services enable redis.googleapis.com
gcloud services enable cloudbuild.googleapis.com
```

### Option B: Using Web Interface (Google Cloud Console)
1. **Go to Google Cloud Console**: https://console.cloud.google.com
2. **Select your project** from the dropdown at the top
3. **Enable APIs one by one**:
   
   **For Cloud Run:**
   - Go to Navigation Menu → Cloud Run
   - Click "Enable Cloud Run API" when prompted
   
   **For Redis (Memorystore):**
   - Go to Navigation Menu → Memorystore → Redis
   - Click "Enable Memorystore for Redis API" when prompted
   
   **For Cloud Build:**
   - Go to Navigation Menu → Cloud Build
   - Click "Enable Cloud Build API" when prompted

4. **Alternative: Use API Library**:
   - Go to Navigation Menu → APIs & Services → Library
   - Search for each service and click "Enable"

## Step 2: Create Redis Instance

### Option A: Using Terminal
```bash
# Create Redis instance in Hong Kong region (takes 5-10 minutes)
gcloud redis instances create askql-redis \
    --size=1 \
    --region=asia-east2 \
    --redis-version=redis_7_0 \
    --tier=basic

# Get Redis connection details
gcloud redis instances describe askql-redis --region=asia-east2
```

### Option B: Using Web Interface
1. **Go to Memorystore**: Navigation Menu → Memorystore → Redis
2. **Click "Create Instance"**
3. **Configure your Redis instance**:
   - Instance ID: `askql-redis`
   - **Region**: `asia-east2` (Hong Kong) or `asia-southeast1` (Singapore) for better latency
   - Redis version: `7.0`
   - **Memory size options**:
     - Basic: 1GB - 300GB
     - Standard (with HA): 5GB - 300GB
   - **Tier**: 
     - Basic (single node, cheaper)
     - Standard (high availability with failover)
   - Network: `default`
   - **Read replicas**: Only available in Standard tier (optional)
4. **Click "Create"** (takes 5-10 minutes)
5. **Get connection details**: Click on your instance name to see IP and port

## Redis Capacity & Features Explained

### Memory Capacity
- **Basic Tier**: 1GB to 300GB
  - Single Redis node
  - Good for development/testing
  - Lower cost
  
- **Standard Tier**: 5GB to 300GB
  - Primary + replica for high availability
  - Automatic failover
  - Production-ready

### Read Replicas (Standard Tier Only)
- **What they are**: Additional read-only copies of your Redis data
- **Purpose**: 
  - Scale read operations (your app can read from multiple replicas)
  - Reduce latency by placing replicas closer to users
  - Increase availability
- **Use case**: When you have heavy read workloads
- **Cost**: Each replica costs the same as the primary instance

### Hong Kong Pricing Impact
**Good news**: Asia-Pacific regions like Hong Kong have competitive pricing!

**Estimated Monthly Costs (Hong Kong region)**:
- **Basic 1GB**: ~$45-55 USD/month
- **Standard 5GB**: ~$180-220 USD/month  
- **Read replica**: +$45-55 USD each

**Cost optimization for your project**:
- Start with **Basic 1GB** in `asia-east2` (Hong Kong)
- Your CSV data is likely small, 1GB is plenty
- Upgrade to Standard only when you need production reliability

## Step 3: Deploy Backend to Cloud Run

### Option A: Using Terminal
```bash
# Navigate to backend directory
cd backend

# Build and deploy to Cloud Run
gcloud run deploy askql-backend \
    --source . \
    --platform managed \
    --region asia-east2 \
    --allow-unauthenticated \
    --set-env-vars REDIS_URL="redis://REDIS_IP:6379"

# Get the service URL
gcloud run services describe askql-backend --region=asia-east2 --format="value(status.url)"
```

### Option B: Using Web Interface
1. **Go to Cloud Run**: Navigation Menu → Cloud Run
2. **Click "Create Service"**
3. **Choose "Deploy one revision from source"**
4. **Configure deployment**:
   - Source: Connect to GitHub repository (askql)
   - Build type: Dockerfile
   - Dockerfile location: `/backend/Dockerfile`
   - Service name: `askql-backend`
   - Region: `asia-east2` (Hong Kong) or `asia-southeast1` (Singapore) for better latency
5. **Set environment variables**:
   - Key: `REDIS_URL`
   - Value: `redis://YOUR_REDIS_IP:6379` (from Step 2)
6. **Configure traffic**:
   - Check "Allow unauthenticated invocations"
7. **Click "Create"**
8. **Get service URL**: Copy the URL from the service details page

## Step 4: Update Vercel Frontend
Update your frontend environment variable to point to the Cloud Run URL:
- NEXT_PUBLIC_API_URL=https://your-cloud-run-url

## Costs (Hong Kong Region)
- **Cloud Run**: Pay per request (~$0.40 per million requests) - same globally
- **Redis Basic 1GB**: ~$45-55 USD/month in Hong Kong
- **Redis Standard 5GB**: ~$180-220 USD/month in Hong Kong
- **Total for development**: ~$50-60/month (much cheaper than US/Europe!)

## Why Hong Kong is Great for GCP:
- **Lower costs** than US/Europe regions
- **Excellent connectivity** to Asia-Pacific
- **Good latency** to mainland China, Southeast Asia
- **Same features** as other regions

## Portfolio/Demo Strategy (Smart Cost Management)

### Perfect for Showing Skills Without Ongoing Costs

**Yes, it's completely fair and smart to:**
1. **Deploy everything** to prove you can do it
2. **Test and document** your working system
3. **Take screenshots/recordings** for your portfolio
4. **Shut down** to avoid monthly charges
5. **Redeploy when needed** (for interviews, demos)

### Documentation Strategy
Before shutting down, capture:
- **Screenshots** of working frontend + backend
- **Architecture diagram** showing GCP services used
- **Screen recording** of CSV upload/query flow
- **Code walkthrough** explaining your implementation
- **Deployment process** you followed

### How to Shut Down GCP Resources

#### Delete Redis Instance (Saves ~$50/month)
```bash
# Terminal
gcloud redis instances delete askql-redis --region=asia-east2

# Or Web Interface: Memorystore → Redis → Select instance → Delete
```

#### Delete Cloud Run Service (Saves ~$5-10/month)
```bash
# Terminal  
gcloud run services delete askql-backend --region=asia-east2

# Or Web Interface: Cloud Run → Select service → Delete
```

#### Keep Your Code & Configs
- **GitHub repo** stays as-is (shows your code)
- **Dockerfile** proves you understand containerization
- **deployment configs** show cloud skills
- **Can redeploy in 10 minutes** for interviews

### Redeployment for Interviews
When you need to show it working:
1. **5 minutes**: Redeploy Redis + Cloud Run
2. **Update Vercel**: Point to new backend URL  
3. **Demo ready**: Full working system
4. **Shut down after**: Demo complete, costs stopped

### Portfolio Value
Employers see:
- ✅ **Full-stack development** (Next.js + Go)
- ✅ **Cloud deployment** (GCP services)
- ✅ **Containerization** (Docker)
- ✅ **Session management** (Redis)
- ✅ **Cost consciousness** (smart resource management)

**This approach shows both technical skills AND business sense!**
