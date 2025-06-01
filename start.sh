#!/bin/bash

# Go Message App - Simple Startup Script
# This script starts the entire application with Docker Compose

echo "🚀 Starting Go Message App..."
echo "================================"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker Desktop and try again."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose not found. Please install Docker Compose."
    exit 1
fi

# Stop any existing containers
echo "🧹 Cleaning up existing containers..."
docker-compose down --remove-orphans

# Build and start all services
echo "🏗️  Building and starting services..."
docker-compose up --build -d

# Wait for services to be healthy
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check service health
echo "🔍 Checking service health..."

# Check auth service
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ Auth service is healthy"
else
    echo "❌ Auth service is not responding"
fi

# Check gateway service
if curl -s http://localhost:8081/health > /dev/null; then
    echo "✅ Gateway service is healthy"
else
    echo "❌ Gateway service is not responding"
fi

echo ""
echo "🎉 Go Message App is ready!"
echo "================================"
echo "📱 Web Chat Interface: http://localhost:8081/chat"
echo "🔐 Auth API: http://localhost:8080"
echo "🌐 Gateway API: http://localhost:8081"
echo ""
echo "📋 Useful commands:"
echo "  View logs: docker-compose logs -f"
echo "  Stop app:  docker-compose down"
echo "  Restart:   docker-compose restart"
echo ""
echo "🎯 Open your browser to http://localhost:8081/chat to start chatting!" 