# Stories Service API

[![CI/CD Pipeline](https://github.com/Princekumarofficial/stories-api-golang/actions/workflows/ci.yml/badge.svg)](https://github.com/Princekumarofficial/stories-api-golang/actions/workflows/ci.yml)
[![Security Audit](https://github.com/Princekumarofficial/stories-api-golang/actions/workflows/security.yml/badge.svg)](https://github.com/Princekumarofficial/stories-api-golang/actions/workflows/security.yml)

A modern Go-based ephemeral stories sharing service with JWT authentication, real-time WebSocket events, and media uploads. Similar to Instagram/Snapchat stories with automatic expiration functionality.

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client App    │    │   WebSocket      │    │   MinIO S3      │
│   (Web/Mobile)  │◄──►│   Real-time      │    │   (Media)       │
└─────────────────┘    │   Events         │    └─────────────────┘
         │              └──────────────────┘             ▲
         │                       │                       │
         │              ┌─────────────────┐              │
         └─────────────►│  Stories API    │──────────────┘
                        │  (Go Service)   │
                        └─────────────────┘
                                 │
                   ┌─────────────┼─────────────┐
                   │             │             │
          ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
          │   PostgreSQL    │ │     Redis       │ │ Ephemeral       │
          │   Database      │ │   (Cache)       │ │ Worker          │
          └─────────────────┘ └─────────────────┘ └─────────────────┘
```

### 🧩 Core Components

- **Stories API**: Go HTTP server with middleware (auth, rate limiting, CORS)
- **Authentication**: JWT-based auth with secure token validation
- **Database**: PostgreSQL for persistent data storage
- **Cache Layer**: Redis for optimized feeds and caching
- **Media Storage**: MinIO S3-compatible object storage
- **Real-time Events**: WebSocket hub for live notifications
- **Background Worker**: Automated story expiration cleanup
- **API Documentation**: Swagger/OpenAPI auto-generated docs

## 🚀 Quick Setup

### Option 1: 🏭 Production Deployment (Recommended)

**Using published Docker images from GitHub Container Registry:**

```bash
# Clone repository
git clone https://github.com/Princekumarofficial/stories-api-golang.git
cd stories-api-golang

# Deploy with one command using pre-built images
./deploy.sh production latest
```

**That's it!** 🎉 All services will be running with:
- **Stories API**: `http://localhost:8080`
- **API Documentation**: `http://localhost:8080/docs/`
- **MinIO Console**: `http://localhost:9001` (minioadmin/minioadmin)

### Option 2: 🛠️ Development Setup

**For local development and building from source:**

```bash
# Clone repository
git clone https://github.com/Princekumarofficial/stories-api-golang.git
cd stories-api-golang

# Copy environment file and start all services
cp .env.example .env
docker-compose up -d
```

### Prerequisites
- Docker & Docker Compose
- Git
- Go 1.21+ (for development only)

**Service URLs:**
- **API Server**: `http://localhost:8080`
- **API Documentation**: `http://localhost:8080/docs/`
- **MinIO Console**: `http://localhost:9001` (minioadmin/minioadmin)
- **WebSocket Test**: Open `tests/websocket-test.html` in browser

## 🎯 Available Deployment Methods

| Method | Use Case | Command | Build Time |
|--------|----------|---------|------------|
| **Production (GHCR)** | 🏭 Production/Staging | `./deploy.sh production latest` | ~10 seconds |
| **Development** | 🛠️ Local development | `docker-compose up -d` | ~2-3 minutes |
| **Local Build** | 🔧 Custom builds | `docker-compose -f docker-compose.local.yml up --build` | ~3-5 minutes |

**Recommended**: Use the **Production (GHCR)** method for fastest deployment with pre-built, tested images.
## 📖 Complete API Walkthrough

### 1. 👤 Sign Up & Login → JWT

#### Sign Up a New User
```bash
curl -X POST http://localhost:8080/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

#### Login and Get JWT Token
```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com", 
    "password": "password123"
  }'
```

**Response (Save the token):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 2. 📁 Get Presigned URL → Upload Media

#### Step 1: Generate Upload URL
```bash
export JWT_TOKEN="your_jwt_token_here"

curl -X POST http://localhost:8080/media/upload-url \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content_type": "image/jpeg"
  }'
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "object_key": "users/12345/media/550e8400-e29b-41d4-a716-446655440000.jpg",
    "upload_url": "http://localhost:9000/stories-media/users/12345/media/...",
    "expires_at": 1640995200,
    "max_file_size": 10485760,
    "content_type": "image/jpeg"
  }
}
```

#### Step 2: Upload File to MinIO
```bash
export UPLOAD_URL="presigned_upload_url_from_response"

curl -X PUT "$UPLOAD_URL" \
  -H "Content-Type: image/jpeg" \
  --data-binary @/path/to/your/image.jpg
```

#### Step 3: Verify Upload
```bash
curl -X GET http://localhost:8080/media \
  -H "Authorization: Bearer $JWT_TOKEN"
```

### 3. 📝 Create a Story (Public/Friends)

#### Create Public Story
```bash
export MEDIA_KEY="users/12345/media/550e8400-e29b-41d4-a716-446655440000.jpg"

curl -X POST http://localhost:8080/stories \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "My amazing public story! 🎉",
    "media_key": "'$MEDIA_KEY'",
    "visibility": "PUBLIC",
    "audience_user_ids": []
  }'
```

#### Create Friends-Only Story  
```bash
curl -X POST http://localhost:8080/stories \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Private moment with friends 👥",
    "media_key": "'$MEDIA_KEY'",
    "visibility": "FRIENDS", 
    "audience_user_ids": ["friend-uuid-1", "friend-uuid-2"]
  }'
```

**Response (Save story_id):**
```json
{
  "status": "success",
  "message": "Story created successfully",
  "story_id": "story-uuid-here"
}
```

### 4. 👥 Follow Test User & Hit `/feed`

#### Follow Another User
```bash
export FRIEND_USER_ID="another-user-uuid"

curl -X POST http://localhost:8080/follow/$FRIEND_USER_ID \
  -H "Authorization: Bearer $JWT_TOKEN"
```

#### Get Your Personalized Feed

`GET /feed` returns up to 50 visible, unexpired stories in `data`, newest first.
Set `limit` to a value from 1 to 100 to change the page size. When the response
includes `next_cursor`, pass it as `cursor` on the next request. The final
page omits `next_cursor`. Stories created at the same time are ordered by
descending story ID.

```bash
# Regular feed
curl -X GET http://localhost:8080/feed \
  -H "Authorization: Bearer $JWT_TOKEN"

# Optimized cached feed (faster)
curl -X GET http://localhost:8080/feed/optimized \
  -H "Authorization: Bearer $JWT_TOKEN"
```

### 5. 👀 View + React → Observe Real-time Events

#### Step 1: Connect to WebSocket (Real-time)
Open `tests/websocket-test.html` in your browser, or use JavaScript:

```javascript
const ws = new WebSocket(`ws://localhost:8080/ws?token=${JWT_TOKEN}`);

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Real-time event:', data);
  
  if (data.type === 'story.viewed') {
    console.log(`👀 ${data.data.viewer_id} viewed your story`);
  } else if (data.type === 'story.reacted') {
    console.log(`${data.data.emoji} ${data.data.user_id} reacted to your story`);
  }
};
```

#### Step 2: View a Story (Triggers Real-time Event)
```bash
export STORY_ID="story-uuid-from-step-3"

curl -X POST http://localhost:8080/stories/$STORY_ID/view \
  -H "Authorization: Bearer $JWT_TOKEN"
```

#### Step 3: React to Story (Triggers Real-time Event)
```bash
curl -X POST http://localhost:8080/stories/$STORY_ID/reactions \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "emoji": "❤️"
  }'
```

**WebSocket Event Received:**
```json
{
  "type": "story.reacted",
  "data": {
    "story_id": "story-uuid-here",
    "user_id": "user-uuid-here", 
    "emoji": "❤️",
    "reacted_at": "2024-01-01T12:00:00Z"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 6. ⚙️ Run Worker → See Expirations in Logs

#### Start Background Worker
```bash
# In a new terminal window
CONFIG_PATH=config/local.yaml go run cmd/ephemeral-worker/main.go
```

**Watch logs for story expiration cleanup:**
```bash
# You'll see logs like:
{"level":"INFO","msg":"Starting ephemeral worker","interval":"1m0s"}
{"level":"INFO","msg":"Cleaning up expired stories","expired_count":3}
{"level":"INFO","msg":"Expired stories cleanup completed","processed":3}
```

#### Monitor Worker Activity
```bash
# Check worker binary if built
ls -la bin/
./bin/ephemeral-worker  # If using production build
```

### 7. 📊 Open `/metrics` and Monitoring Dashboard

#### Cache Statistics
```bash
curl -X GET http://localhost:8080/cache/stats \
  -H "Authorization: Bearer $JWT_TOKEN"
```

#### Health Check
```bash
curl -X GET http://localhost:8080/
# Response: "Hello, World!"
```

#### User Statistics  
```bash
curl -X GET http://localhost:8080/me/stats \
  -H "Authorization: Bearer $JWT_TOKEN"
```

#### API Documentation
Open your browser: **http://localhost:8080/docs/**

#### Clear Cache (Development)
```bash
curl -X DELETE http://localhost:8080/cache/clear \
  -H "Authorization: Bearer $JWT_TOKEN"
```

#### Monitoring Services
- **MinIO Console**: http://localhost:9001 (minioadmin/minioadmin)
- **Redis**: localhost:6379 (use redis-cli to monitor)
- **PostgreSQL**: localhost:5432 (use psql to inspect)

## 📋 Complete API Reference

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| **Authentication** |
| POST | `/signup` | User registration | ❌ |
| POST | `/login` | User authentication | ❌ |
| **Stories** |
| POST | `/stories` | Create new story | ✅ |
| GET | `/stories/{id}` | Get specific story | ✅ |
| GET | `/feed` | Get personalized feed | ✅ |
| GET | `/feed/optimized` | Get cached optimized feed | ✅ |
| POST | `/stories/{id}/view` | View story (triggers real-time event) | ✅ |
| POST | `/stories/{id}/reactions` | React to story (triggers real-time event) | ✅ |
| **Social** |
| POST | `/follow/{user_id}` | Follow user | ✅ |
| DELETE | `/follow/{user_id}` | Unfollow user | ✅ |
| GET | `/me/stats` | Get user statistics | ✅ |
| **Media** |
| POST | `/media/upload-url` | Generate presigned upload URL | ✅ |
| GET | `/media` | List user's media files | ✅ |
| GET | `/media/{object_key}/info` | Get media file info | ✅ |
| GET | `/media/{object_key}/download-url` | Generate download URL | ✅ |
| DELETE | `/media/{object_key}` | Delete media file | ✅ |
| **Real-time** |
| GET | `/ws` | WebSocket connection for events | ✅ |
| **Monitoring** |
| GET | `/` | Health check | ❌ |
| GET | `/cache/stats` | Cache statistics | ❌ |
| DELETE | `/cache/clear` | Clear cache (dev only) | ❌ |
| GET | `/docs/` | Swagger API documentation | ❌ |

## 🗄️ Data Models & Storage

### Database Schema (PostgreSQL)
```sql
-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Stories table  
CREATE TABLE stories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id UUID REFERENCES users(id) ON DELETE CASCADE,
    text TEXT,
    media_key VARCHAR(255),
    visibility VARCHAR(20) CHECK (visibility IN ('PUBLIC', 'FRIENDS', 'PRIVATE')),
    audience_user_ids UUID[],
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP DEFAULT (NOW() + INTERVAL '24 hours'),
    deleted_at TIMESTAMP
);

-- Follows table
CREATE TABLE follows (
    follower_id UUID REFERENCES users(id) ON DELETE CASCADE,
    followed_id UUID REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (follower_id, followed_id)
);

-- Story views table  
CREATE TABLE story_views (
    story_id UUID REFERENCES stories(id) ON DELETE CASCADE,
    viewer_id UUID REFERENCES users(id) ON DELETE CASCADE,
    viewed_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (story_id, viewer_id)
);

-- Story reactions table
CREATE TABLE story_reactions (
    story_id UUID REFERENCES stories(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    emoji VARCHAR(10) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (story_id, user_id)
);
```

### Media Storage (MinIO S3)
- **Path Structure**: `users/{user_id}/media/{uuid}.{ext}`
- **Supported Types**: JPEG, PNG, GIF, MP4, MPEG
- **Max Size**: 10MB per file
- **Security**: User-isolated paths, presigned URLs

### Cache Layer (Redis)
- **Feed Caching**: Optimized personalized feeds
- **Query Caching**: Frequently accessed data
- **Session Storage**: Optional JWT blacklisting

## 🔧 Development & Production

### Project Structure
```
├── cmd/
│   ├── stories-service/         # Main API server
│   └── ephemeral-worker/        # Background worker for cleanup
├── config/
│   ├── local.yaml              # Development configuration
│   └── production.yaml         # Production configuration
├── internal/
│   ├── cache/                  # Redis caching layer
│   ├── config/                 # Configuration loading
│   ├── events/                 # Real-time event publishing
│   ├── http/
│   │   ├── handlers/           # HTTP request handlers
│   │   └── middleware/         # Auth, rate limiting middleware
│   ├── services/               # Business logic services
│   ├── storage/                # Database abstraction layer
│   ├── types/                  # Data models and types
│   ├── utils/                  # JWT, password utilities
│   └── websocket/              # Real-time WebSocket hub
├── docs/                       # API documentation
├── tests/                      # Test files and utilities
└── storage/                    # Local storage (development)
```

### Building for Production
```bash
# Build all binaries
./build.sh

# Or build individually
go build -o bin/stories-service cmd/stories-service/main.go
go build -o bin/ephemeral-worker cmd/ephemeral-worker/main.go

# Run in production
./bin/stories-service
./bin/ephemeral-worker
```

## 🚀 Deployment Options

### Option 1: 🏭 Production Deployment (GitHub Container Registry)

**Quick deployment using pre-built Docker images:**

```bash
# Deploy latest version
./deploy.sh production latest

# Deploy specific version
./deploy.sh production v1.0.0

# Or manually with docker-compose
sudo docker compose -f docker-compose.production.yml up -d
```

**Available Images:**
- `ghcr.io/princekumarofficial/stories-api-golang/stories-service:latest`
- `ghcr.io/princekumarofficial/stories-api-golang/ephemeral-worker:latest`

**Features:**
- ✅ Pre-built optimized images from CI/CD
- ✅ Automatic health checks and monitoring  
- ✅ Redis caching included
- ✅ Production-ready configuration
- ✅ One-command deployment

### Option 2: 🛠️ Local Development Build

**Building from source for development:**

```bash
# Start development environment
docker-compose up -d

# Or build manually
./build.sh
```

### Option 3: 📦 Manual Docker Build

**For custom deployments:**

```bash
# Production deployment (builds locally)
docker-compose -f docker-compose.local.yml up --build -d

# Development environment
docker-compose up -d
```

## ⚙️ Configuration & Management

### Production Environment
The production deployment includes:
- **Database**: PostgreSQL 15
- **Cache**: Redis 7  
- **Storage**: MinIO S3-compatible storage
- **Config**: `config/production.yaml` (mounted as volume)

### Environment Variables (Production)
```bash
export CONFIG_PATH="/app/config/production.yaml"
export STORIES_VERSION="latest"  # Or specific version like "v1.0.0"
export WORKER_VERSION="latest"
```

## 🔄 CI/CD Pipeline

**Automated GitHub Actions:**
- ✅ **Build & Test**: Runs on every push/PR
- ✅ **Multi-platform**: AMD64 and ARM64 support  
- ✅ **Security Scanning**: Trivy vulnerability scanning
- ✅ **Auto-publishing**: Pushes to GitHub Container Registry
- ✅ **Dependency Updates**: Automated with Dependabot

**Image Tags:**
- `latest` - Latest stable from main branch
- `v1.0.0` - Specific version releases  
- `sha-abc1234` - Specific commit builds
- `develop` - Development branch builds

## 📊 Monitoring & Management

```bash
# Check container status
sudo docker compose -f docker-compose.production.yml ps

# View logs
sudo docker compose -f docker-compose.production.yml logs -f

# Health check
curl http://localhost:8080/health

# Stop services  
sudo docker compose -f docker-compose.production.yml down
```

## � Troubleshooting

### Common Issues

**1. Containers restarting:**
```bash
# Check logs
sudo docker compose -f docker-compose.production.yml logs stories-service
sudo docker compose -f docker-compose.production.yml logs ephemeral-worker
```

**2. Permission issues with Docker:**
```bash
# Add user to docker group (one-time setup)
sudo usermod -aG docker $USER
newgrp docker

# Or use sudo with commands
sudo docker compose -f docker-compose.production.yml up -d
```

**3. Port conflicts:**
```bash
# Check what's using the ports
sudo netstat -tulpn | grep :8080
sudo netstat -tulpn | grep :5432

# Stop conflicting services or change ports in config
```

**4. Images not pulling:**
```bash
# Login to GitHub Container Registry (if needed)
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Or pull manually
docker pull ghcr.io/princekumarofficial/stories-api-golang/stories-service:latest
```

## �🔒 Security Features

- ✅ **JWT Authentication**: Secure token-based auth
- ✅ **Password Hashing**: bcrypt for secure password storage
- ✅ **Input Validation**: Request validation and sanitization
- ✅ **SQL Injection Prevention**: Parameterized queries
- ✅ **CORS Configuration**: Cross-origin request handling
- ✅ **Rate Limiting**: API endpoint protection
- ✅ **Media Security**: User-isolated storage paths
- ✅ **WebSocket Auth**: JWT-secured real-time connections

## 🚀 Performance & Scalability

- ✅ **Redis Caching**: Optimized database queries
- ✅ **Connection Pooling**: Efficient database connections
- ✅ **Background Processing**: Non-blocking story expiration
- ✅ **Real-time Events**: Efficient WebSocket hub
- ✅ **Optimized Feeds**: Cached personalized content
- ✅ **MinIO Storage**: Scalable object storage
- ✅ **Docker Ready**: Containerized deployment

## 📚 Documentation & Testing

- **API Docs**: http://localhost:8080/docs/ (Swagger/OpenAPI)
- **WebSocket Test**: `tests/websocket-test.html` 
- **Real-time Events**: `docs/websocket-events.md`
- **Performance**: `PERFORMANCE_CACHING.md`
- **Implementation**: `REALTIME_EVENTS_IMPLEMENTATION.md`
- **CI/CD Setup**: `CI_CD_README.md` (Complete CI/CD pipeline documentation)

---

**Built with ❤️ using Go, PostgreSQL, Redis, MinIO, and WebSocket technology.**

*A modern, scalable stories platform with real-time features, enterprise-grade security, and automated CI/CD.*
