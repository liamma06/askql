# Deploy to Railway (Easiest Option)

## Why Railway?
- GitHub integration (auto-deploy on push)
- Built-in Redis addon
- Very simple setup
- Great for portfolios

## Setup Steps

### 1. Create Railway Account
- Go to https://railway.app
- Sign up with GitHub

### 2. Deploy Backend
1. Click "New Project" → "Deploy from GitHub repo"
2. Select your askql repository
3. Railway will auto-detect the Dockerfile
4. Set root directory to `/backend`
5. Add environment variables:
   - `PORT=8080`
   - `REDIS_URL=redis://localhost:6379` (temporary)

### 3. Add Redis
1. In your Railway project dashboard
2. Click "New" → "Database" → "Add Redis"
3. Copy the Redis connection URL
4. Update your backend service environment:
   - `REDIS_URL=redis://default:password@host:port`

### 4. Get Backend URL
- Railway will provide a URL like: `https://askql-backend-production.up.railway.app`

### 5. Update Vercel
- Set `NEXT_PUBLIC_API_URL` to your Railway backend URL

## Costs
- $5/month for hobby plan (includes backend + Redis)
- Perfect for portfolios and learning
