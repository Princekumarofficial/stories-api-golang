# Stories Service API

A modern Go-based stories sharing service with JWT authentication, real-time features, and media uploads. Similar to Instagram/Snapchat stories with expiration functionality.

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client App    │    │   Load Balancer  │    │   CDN/S3        │
│   (Web/Mobile)  │◄──►│   (Optional)     │    │   (Media)       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       ▲
         │                       ▼                       │
         │              ┌─────────────────┐              │
         └─────────────►│  Stories API    │──────────────┘
                        │  (Go Service)   │
                        └─────────────────┘
                                 │
                        ┌─────────────────┐
                        │   PostgreSQL    │
                        │   Database      │
                        └─────────────────┘
```

### 🧩 Core Components

- **HTTP Server**: Go standard library with custom middleware
- **Authentication**: JWT-based auth with middleware protection
- **Database**: PostgreSQL with direct SQL queries
- **API Documentation**: Swagger/OpenAPI integration
- **Configuration**: YAML-based config management
- **Media Handling**: Pre-signed URL generation for uploads
- **Story Expiration**: Background worker for cleanup
- **Real-time Events**: WebSocket/SSE for live updates

## 🚀 Quick Setup

### Prerequisites
- Go 1.24.2+
- PostgreSQL 15+
- Docker & Docker Compose (optional)

### Environment Configuration

Create a `.env` file in the project root:

```env
# Database Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=password123
POSTGRES_DB=stories_db

# JWT Configuration
JWT_SECRET=your_super_secret_jwt_key_here_change_in_production

# Server Configuration
HTTP_ADDRESS=localhost:8080

# Environment
ENV=development

# Media Storage (if using cloud storage)
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
S3_BUCKET_NAME=your-stories-bucket
```

### 🐳 One-Command Run (Docker)

```bash
# Clone and start everything
git clone --
cd GOLANG-API
cp .env.example .env  # Edit with your values
docker-compose up -d postgres
go run cmd/stories-service/main.go
```

### 🔧 Manual Setup

```bash
# 1. Start PostgreSQL and MinIO
docker-compose up -d

# 2. Install dependencies
go mod tidy

# 3. Run database migrations (if available)
# go run cmd/migrate/main.go up

# 4. Start the service
CONFIG_PATH=config/local.yaml go run cmd/stories-service/main.go
```

The service will be available at:
- **API Server**: `http://localhost:8080`
- **MinIO Console**: `http://localhost:9001` (admin/admin)
- **MinIO API**: `http://localhost:9000`

## 🎯 Media Upload Configuration

### MinIO Setup
The application uses MinIO for object storage with the following default configuration:

- **Endpoint**: `localhost:9000`
- **Access Key**: `minioadmin`
- **Secret Key**: `minioadmin`
- **Bucket**: `stories-media`
- **Console**: `http://localhost:9001`

### Supported Media Types
- **Images**: `image/jpeg`, `image/png`, `image/gif`
- **Videos**: `video/mp4`, `video/mpeg`
- **Max File Size**: 10MB (configurable)
- **Upload URL TTL**: 1 hour (configurable)

### Security Features
- ✅ Content-type validation
- ✅ File size limits
- ✅ User-isolated storage paths
- ✅ Presigned URL expiration
- ✅ JWT authentication required
- ✅ Object ownership verification

## 📖 API Walkthrough

### 1. 👤 Sign Up & Login → JWT

#### Sign Up
```bash
curl -X POST http://localhost:8080/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

#### Login
```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user_id": "user-uuid-here"
}
```

### 2. 📁 Media Upload with MinIO

#### Generate Upload URL
```bash
# Get presigned upload URL
curl -X POST http://localhost:8080/media/upload-url \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content_type": "image/jpeg"
  }'
```

**Response:**
```json
{
  "status": "success",
  "message": "Upload URL generated successfully",
  "data": {
    "object_key": "users/12345/media/550e8400-e29b-41d4-a716-446655440000.jpg",
    "upload_url": "http://localhost:9000/stories-media/users/12345/media/550e8400-e29b-41d4-a716-446655440000.jpg?...",
    "expires_at": 1640995200,
    "max_file_size": 10485760,
    "content_type": "image/jpeg"
  }
}
```

#### Upload File to MinIO
```bash
# Upload file using the presigned URL
curl -X PUT "PRESIGNED_UPLOAD_URL" \
  -H "Content-Type: image/jpeg" \
  --data-binary @path/to/your/image.jpg
```

#### List User Media
```bash
# Get all media files for authenticated user
curl -X GET http://localhost:8080/media \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**Response:**
```json
{
  "status": "success",
  "message": "Media files retrieved successfully",
  "data": [
    {
      "object_key": "users/12345/media/550e8400-e29b-41d4-a716-446655440000.jpg",
      "size": 1024000,
      "content_type": "image/jpeg",
      "uploaded_at": "2024-01-01T12:00:00Z",
      "media_url": "http://localhost:9000/stories-media/users/12345/media/550e8400-e29b-41d4-a716-446655440000.jpg"
    }
  ]
}
```

#### Get Media Info
```bash
# Get information about a specific media file
curl -X GET "http://localhost:8080/media/users%2F12345%2Fmedia%2F550e8400-e29b-41d4-a716-446655440000.jpg/info" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Generate Download URL
```bash
# Get presigned download URL
curl -X GET "http://localhost:8080/media/users%2F12345%2Fmedia%2F550e8400-e29b-41d4-a716-446655440000.jpg/download-url?expires=3600" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### 2. 📁 Get Presigned URL → Upload Media (Legacy)

```bash
# Get upload URL (replace with actual endpoint when implemented)
curl -X POST http://localhost:8080/media/upload-url \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "file_type": "image/jpeg",
    "file_size": 1024000
  }'
```

**Response:**
```json
{
  "upload_url": "https://s3.amazonaws.com/bucket/upload-url",
  "media_key": "media-uuid-key"
}
```

### 3. 📝 Create a Story (Public/Friends)

```bash
# Create a public story
curl -X POST http://localhost:8080/stories \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "My amazing story!",
    "media_key": "media-uuid-key",
    "visibility": "public",
    "audience_user_ids": []
  }'

# Create a friends-only story
curl -X POST http://localhost:8080/stories \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Private moment with friends",
    "media_key": "media-uuid-key",
    "visibility": "friends",
    "audience_user_ids": ["friend-uuid-1", "friend-uuid-2"]
  }'
```

### 4. 👥 Follow a Test User & Hit `/feed`

```bash
# Follow a user (endpoint to be implemented)
curl -X POST http://localhost:8080/follow \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test-user-uuid"
  }'

# Get your personalized feed
curl -X GET http://localhost:8080/feed \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### 5. 👀 View + React → Real-time Events

```bash
# View a story
curl -X POST http://localhost:8080/stories/STORY_ID/view \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# React to a story
curl -X POST http://localhost:8080/stories/STORY_ID/react \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reaction_type": "like"
  }'
```

**WebSocket Connection for Real-time:**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws?token=YOUR_JWT_TOKEN');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Real-time event:', data);
};
```

### 6. ⚙️ Run Worker → See Expirations in Logs

```bash
# Start the background worker (if implemented as separate service)
go run cmd/worker/main.go

# Or check logs from main service for expiration cleanup
tail -f logs/stories-service.log | grep "expired"
```

### 7. 📊 Open `/metrics` and Kibana Dashboard

```bash
# Prometheus metrics endpoint
curl http://localhost:8080/metrics

# Health check
curl http://localhost:8080/health
```

**Grafana Dashboard** (if configured):
- URL: `http://localhost:3000`
- Default login: `admin/admin`

## 📋 API Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/` | Health check | ❌ |
| POST | `/signup` | User registration | ❌ |
| POST | `/login` | User authentication | ❌ |
| GET | `/feed` | Get stories feed | ❌ |
| POST | `/stories` | Create new story | ✅ |
| GET | `/swagger/` | API documentation | ❌ |

## 🗄️ Database Schema

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
    visibility VARCHAR(20) CHECK (visibility IN ('public', 'friends', 'private')),
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP DEFAULT (NOW() + INTERVAL '24 hours'),
    deleted_at TIMESTAMP
);
```

## 🔧 Development

### Project Structure
```
├── cmd/
│   └── stories-service/          # Main application entry point
├── config/                       # Configuration files
├── docs/                        # Swagger documentation
├── internal/
│   ├── config/                  # Configuration loading
│   ├── http/
│   │   ├── handlers/           # HTTP request handlers
│   │   └── middleware/         # Authentication middleware
│   ├── storage/                # Database layer
│   ├── types/                  # Data models
│   └── utils/                  # Utility functions
└── storage/                    # SQLite file (if using)
```

### Building for Production
```bash
# Build binary
go build -o bin/stories-service cmd/stories-service/main.go

# Run binary
./bin/stories-service
```

## 🔒 Security Features

- ✅ JWT-based authentication
- ✅ Password hashing with bcrypt
- ✅ Request validation
- ✅ SQL injection prevention
- ✅ CORS configuration
- ✅ Rate limiting (middleware ready)
- ✅ Input sanitization

```

### Environment Variables
```bash
export CONFIG_PATH="/app/config/production.yaml"
export JWT_SECRET="production-secret-key"
export POSTGRES_HOST="production-db-host"
```


**Made with ❤️ using Go, PostgreSQL, and modern web technologies.**
