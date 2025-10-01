#!/bin/bash

# Deployment script for Stories API
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
ENVIRONMENT=${1:-production}
VERSION=${2:-latest}
REGISTRY="ghcr.io/princekumarofficial/stories-api-golang"

echo -e "${GREEN}🚀 Deploying Stories API${NC}"
echo -e "${YELLOW}Environment: ${ENVIRONMENT}${NC}"
echo -e "${YELLOW}Version: ${VERSION}${NC}"
echo -e "${YELLOW}Registry: ${REGISTRY}${NC}"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker first.${NC}"
    exit 1
fi

# Pull latest images
echo -e "${GREEN}📥 Pulling Docker images...${NC}"
docker pull ${REGISTRY}/stories-service:${VERSION}
docker pull ${REGISTRY}/ephemeral-worker:${VERSION}

# Stop existing containers
echo -e "${YELLOW}🛑 Stopping existing containers...${NC}"
docker-compose -f docker-compose.production.yml down || true

# Start new containers
echo -e "${GREEN}🚀 Starting containers...${NC}"
export STORIES_VERSION=${VERSION}
export WORKER_VERSION=${VERSION}

docker-compose -f docker-compose.production.yml up -d

# Wait for services to be healthy
echo -e "${YELLOW}⏳ Waiting for services to be ready...${NC}"
sleep 10

# Health check
echo -e "${GREEN}🔍 Checking service health...${NC}"
if curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Stories service is healthy${NC}"
else
    echo -e "${RED}❌ Stories service health check failed${NC}"
    docker-compose -f docker-compose.production.yml logs stories-service
    exit 1
fi

echo -e "${GREEN}🎉 Deployment completed successfully!${NC}"
echo -e "${YELLOW}Services running at:${NC}"
echo -e "  - Stories API: http://localhost:8080"
echo -e "  - MinIO Console: http://localhost:9001"

# Show running containers
echo -e "${GREEN}📊 Running containers:${NC}"
docker-compose -f docker-compose.production.yml ps
