# 🚀 Go Message App - Real-Time Microservices Chat Application

A production-ready, event-driven real-time messaging application built with Go microservices architecture, featuring WebSocket communication, JWT authentication, and message persistence.

## 🏗️ **Architecture Overview**

This application consists of 5 microservices:
- **🔐 Auth Service** (Port 8080): User registration, login, JWT token management
- **🌐 Gateway Service** (Port 8081): WebSocket chat gateway + Web UI
- **💾 Persist Service**: Kafka consumer for message persistence to PostgreSQL
- **🗄️ Database**: PostgreSQL for user data and message storage
- **📨 Kafka**: Apache Kafka message broker for real-time event streaming

## 🛠️ **Technologies Used**

- **Backend**: Go (Golang)
- **Database**: PostgreSQL
- **Message Broker**: Apache Kafka (KRaft mode)
- **Authentication**: JWT tokens
- **Communication**: WebSockets, REST APIs
- **Containerization**: Docker
- **Orchestration**: Kubernetes (kind cluster)
- **Frontend**: HTML5, JavaScript, WebSocket API

## 📋 **Prerequisites**

Before starting, ensure you have the following installed on your system:

### **Required Software**

1. **Go** (version 1.19+)
   ```bash
   # macOS
   brew install go
   
   # Or download from: https://golang.org/dl/
   ```

2. **Docker** (version 20.0+)
   ```bash
   # macOS
   brew install --cask docker
   
   # Or download Docker Desktop from: https://www.docker.com/products/docker-desktop
   ```

3. **kubectl** (Kubernetes CLI)
   ```bash
   # macOS
   brew install kubectl
   
   # Or follow: https://kubernetes.io/docs/tasks/tools/install-kubectl/
   ```

4. **kind** (Kubernetes in Docker)
   ```bash
   # macOS
   brew install kind
   
   # Or follow: https://kind.sigs.k8s.io/docs/user/quick-start/#installation
   ```

5. **Git**
   ```bash
   # macOS
   brew install git
   ```

### **Optional Tools for Testing**

```bash
# WebSocket testing tool
npm install -g wscat

# Advanced WebSocket tool
brew install websocat

# JSON processing
brew install jq
```

## 🚀 **Quick Start Guide**

### **Step 1: Clone the Repository**

```bash
git clone https://github.com/saim61/go_message_app.git
cd go_message_app
```

### **Step 2: Create and Configure kind Cluster**

```bash
# Create a kind cluster
kind create cluster --name go-message-app

# Verify cluster is running
kubectl cluster-info --context kind-go-message-app
```

### **Step 3: Build and Load Docker Images**

```bash
# Build all microservice images
docker build --build-arg SERVICE=auth -t go-message-app-auth:latest .
docker build --build-arg SERVICE=gateway -t go-message-app-gateway:latest .
docker build --build-arg SERVICE=persist -t go-message-app-persist:latest .

# Load images into kind cluster
kind load docker-image go-message-app-auth:latest --name go-message-app
kind load docker-image go-message-app-gateway:latest --name go-message-app
kind load docker-image go-message-app-persist:latest --name go-message-app
```

### **Step 4: Deploy to Kubernetes**

```bash
# Deploy all services
kubectl apply -f k8s/

# Wait for all pods to be ready (this may take 2-3 minutes)
kubectl get pods -n go-message-app -w
```

**Expected output when ready:**
```
NAME                       READY   STATUS    RESTARTS   AGE
auth-xxxxxxxxx-xxxxx       1/1     Running   0          2m
db-xxxxxxxxx-xxxxx         1/1     Running   0          2m
gateway-xxxxxxxxx-xxxxx    1/1     Running   0          2m
kafka-xxxxxxxxx-xxxxx      1/1     Running   0          2m
persist-xxxxxxxxx-xxxxx    1/1     Running   0          2m
```

### **Step 5: Set Up Database Tables**

```bash
# Create users table
kubectl exec -n go-message-app deployment/db -- \
  psql -U postgres -d go-message-app -c \
  "CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY, 
    username TEXT UNIQUE NOT NULL, 
    password TEXT NOT NULL, 
    created_at TIMESTAMPTZ DEFAULT now()
  );"

# Create messages table
kubectl exec -n go-message-app deployment/db -- \
  psql -U postgres -d go-message-app -c \
  "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"; 
   CREATE TABLE IF NOT EXISTS messages (
     id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), 
     room TEXT NOT NULL, 
     author_id INT NOT NULL REFERENCES users(id), 
     body TEXT NOT NULL, 
     created_at TIMESTAMPTZ DEFAULT now()
   );"

# Verify tables were created
kubectl exec -n go-message-app deployment/db -- \
  psql -U postgres -d go-message-app -c "\dt"
```

### **Step 6: Set Up Port Forwarding**

```bash
# Clean up any existing port forwards
pkill -f "kubectl port-forward" 2>/dev/null || true

# Set up port forwarding for both services
kubectl port-forward -n go-message-app service/auth 8080:8080 &
kubectl port-forward -n go-message-app service/gateway 8081:8081 &

# Verify port forwarding is working
sleep 3
curl -s http://localhost:8080/health
curl -s http://localhost:8081/health
```

**Expected output:** `{"status":"healthy"}` for both services.

## 🎉 **Using the Application**

### **🌟 Web Chat Interface (Recommended)**

1. **Open your browser** and navigate to: **http://localhost:8081/chat**

2. **Register a new user:**
   - Enter a username (e.g., "alice")
   - Enter a password (e.g., "password123")
   - Click "Register"

3. **Login:**
   - Click "Login" with the same credentials
   - Your JWT token will be automatically handled

4. **Start chatting:**
   - Click "Connect to Chat"
   - Type messages and press Enter
   - Open multiple browser tabs to test multi-user chat!

### **🔧 API Testing (Advanced)**

#### **Health Checks**
```bash
# Check auth service
curl http://localhost:8080/health

# Check gateway service
curl http://localhost:8081/health
```

#### **User Registration**
```bash
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "testpass123"
  }'
```

#### **User Login**
```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "testpass123"
  }'
```

#### **WebSocket Chat Testing**
```bash
# Install wscat if not already installed
npm install -g wscat

# Get JWT token and connect to WebSocket
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username": "testuser", "password": "testpass123"}' | \
  jq -r '.data.token')

# Connect to WebSocket
wscat -c "ws://localhost:8081/ws?token=$TOKEN"

# Send a message (after connection is established)
{"room": "general", "body": "Hello, World!"}
```

## 📊 **Monitoring and Debugging**

### **Check Service Status**
```bash
# View all pods
kubectl get pods -n go-message-app

# Check specific service logs
kubectl logs -n go-message-app deployment/auth -f
kubectl logs -n go-message-app deployment/gateway -f
kubectl logs -n go-message-app deployment/persist -f
kubectl logs -n go-message-app deployment/kafka -f
kubectl logs -n go-message-app deployment/db -f
```

### **Monitor Kafka Messages**
```bash
# List Kafka topics
kubectl exec -n go-message-app deployment/kafka -- \
  /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list

# Monitor chat messages in real-time
kubectl exec -n go-message-app deployment/kafka -- \
  /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic chat-in \
  --from-beginning
```

### **Check Database**
```bash
# Connect to PostgreSQL
kubectl exec -it -n go-message-app deployment/db -- \
  psql -U postgres -d go-message-app

# View users
SELECT * FROM users;

# View messages
SELECT * FROM messages;

# Exit with \q
```

## 🛠️ **Development Workflow**

### **Making Code Changes**

1. **Edit your Go code**
2. **Rebuild the specific service:**
   ```bash
   # Example: rebuilding auth service
   docker build --build-arg SERVICE=auth -t go-message-app-auth:latest .
   kind load docker-image go-message-app-auth:latest --name go-message-app
   ```

3. **Restart the deployment:**
   ```bash
   kubectl rollout restart deployment/auth -n go-message-app
   ```

### **Adding New Features**

1. **Create new endpoints** in the appropriate service
2. **Update Kubernetes manifests** if needed
3. **Test locally** using the web interface or API calls
4. **Monitor logs** for any issues

## 🚨 **Troubleshooting**

### **Common Issues and Solutions**

#### **Pods not starting**
```bash
# Check pod status and events
kubectl describe pods -n go-message-app

# Check if images are loaded
docker images | grep go-message-app
```

#### **Port forwarding issues**
```bash
# Kill existing port forwards
pkill -f "kubectl port-forward"

# Restart port forwarding
kubectl port-forward -n go-message-app service/auth 8080:8080 &
kubectl port-forward -n go-message-app service/gateway 8081:8081 &
```

#### **Database connection issues**
```bash
# Check database pod
kubectl logs -n go-message-app deployment/db

# Recreate database tables if needed
kubectl exec -n go-message-app deployment/db -- \
  psql -U postgres -d go-message-app -c "\dt"
```

#### **Kafka issues**
```bash
# Check Kafka logs
kubectl logs -n go-message-app deployment/kafka

# Restart Kafka if needed
kubectl rollout restart deployment/kafka -n go-message-app
```

### **Clean Up and Reset**

#### **Reset the entire application**
```bash
# Delete the namespace (removes all resources)
kubectl delete namespace go-message-app --wait=true

# Recreate everything
kubectl apply -f k8s/
```

#### **Delete kind cluster**
```bash
# Delete the cluster completely
kind delete cluster --name go-message-app
```

#### **Clean up Docker images**
```bash
# Remove all application images
docker rmi go-message-app-auth:latest
docker rmi go-message-app-gateway:latest
docker rmi go-message-app-persist:latest
```

## 📁 **Project Structure**

```
go_message_app/
├── cmd/                    # Application entry points
│   ├── auth/              # Auth service main
│   ├── gateway/           # Gateway service main
│   └── persist/           # Persist service main
├── internal/              # Internal packages
│   ├── auth/             # Authentication logic
│   ├── broker/           # Kafka broker implementation
│   └── database/         # Database connections
├── k8s/                  # Kubernetes manifests
│   ├── 00-namespace.yaml
│   ├── 10-db.yaml
│   ├── 20-kafka.yaml
│   ├── 30-auth.yaml
│   ├── 31-gateway.yaml
│   └── 32-persist.yaml
├── utils/                # Utility functions
├── Dockerfile           # Multi-stage Docker build
├── go.mod              # Go module dependencies
├── go.sum              # Go module checksums
├── README.md           # This file
└── TESTING_GUIDE.md    # Detailed testing instructions
```

## 🎯 **Features**

- ✅ **Real-time messaging** with WebSockets
- ✅ **JWT authentication** with 15-minute token expiration
- ✅ **Room-based chat** for organized conversations
- ✅ **Message persistence** to PostgreSQL database
- ✅ **Event-driven architecture** with Kafka
- ✅ **Microservices design** with independent scaling
- ✅ **Beautiful web interface** with responsive design
- ✅ **Multi-user support** with real-time message broadcasting
- ✅ **Health monitoring** endpoints for all services
- ✅ **Kubernetes deployment** with proper service discovery

## 🔮 **Future Enhancements**

- [ ] Message history retrieval
- [ ] User presence indicators
- [ ] Private messaging
- [ ] File upload support
- [ ] Message reactions/emojis
- [ ] Push notifications
- [ ] Admin panel
- [ ] Rate limiting
- [ ] Message encryption

## 📚 **Additional Resources**

- **Detailed Testing Guide**: See `TESTING_GUIDE.md` for comprehensive testing instructions
- **API Documentation**: All endpoints documented in the testing guide
- **WebSocket Protocol**: JSON-based message format for real-time communication

## 🤝 **Contributing**

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly using the web interface
5. Submit a pull request

## 📄 **License**

This project is open source and available under the [MIT License](LICENSE).

---

## 🎉 **Quick Success Check**

After following the setup instructions, you should be able to:

1. ✅ Open http://localhost:8081/chat in your browser
2. ✅ Register and login with a new user
3. ✅ Connect to the chat and send messages
4. ✅ Open multiple browser tabs and see real-time messaging
5. ✅ Check that all pods are running: `kubectl get pods -n go-message-app`

**🚀 Congratulations! Your Go microservices chat application is now running!**

For detailed testing instructions and advanced usage, see the `TESTING_GUIDE.md` file.