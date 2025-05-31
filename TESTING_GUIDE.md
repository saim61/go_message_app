# 🧪 **Go Message App - Testing Guide**

## 📋 **Service Overview**

Your microservices application consists of:
- **Auth Service** (Port 8080): User registration, login, JWT tokens
- **Gateway Service** (Port 8081): WebSocket chat gateway, **Chat Interface**
- **Persist Service**: Kafka consumer for message persistence
- **Database**: PostgreSQL for user data and messages
- **Kafka**: Message broker for real-time chat

## 🔗 **Access URLs**

With port forwarding active:
- **🌟 Chat Interface**: `http://localhost:8081/chat`
- **Auth Service**: `http://localhost:8080`
- **WebSocket**: `ws://localhost:8081/ws?token=YOUR_JWT_TOKEN`
- **Health Checks**: `http://localhost:8080/health`, `http://localhost:8081/health`

## 🏥 **Health Check Endpoints**

```bash
# Check auth service health
curl http://localhost:8080/health

# Check gateway service health  
curl http://localhost:8081/health
```

Expected response: `{"status":"healthy"}`

## 🔐 **Authentication Endpoints**

### **1. Register a New User**

```bash
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "testpass123"
  }'
```

Expected response:
```json
{
  "success": true,
  "message": "Registration Successful! Please proceed to login to get JWT.",
  "data": {}
}
```

### **2. Login and Get JWT Token**

```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser", 
    "password": "testpass123"
  }'
```

Expected response:
```json
{
  "success": true,
  "message": "Successfully login",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

**💡 Save this token - you'll need it for WebSocket connections!**

## 💬 **WebSocket Chat Testing**

### **Method 1: Using a WebSocket Client Tool**

1. **Install wscat** (WebSocket command line client):
   ```bash
   npm install -g wscat
   ```

2. **Connect to WebSocket** (replace `YOUR_JWT_TOKEN` with the token from login):
   ```bash
   wscat -c "ws://localhost:8081/ws?token=YOUR_JWT_TOKEN"
   ```

3. **Send a message**:
   ```json
   {"room": "general", "body": "Hello, World!"}
   ```

### **Method 2: Using a Simple HTML Client**

Create a test HTML file:

```html
<!DOCTYPE html>
<html>
<head>
    <title>Chat Test</title>
</head>
<body>
    <div id="messages"></div>
    <input type="text" id="messageInput" placeholder="Type a message...">
    <button onclick="sendMessage()">Send</button>
    
    <script>
        // Replace with your actual JWT token
        const token = "YOUR_JWT_TOKEN_HERE";
        const ws = new WebSocket(`ws://localhost:8081/ws?token=${token}`);
        
        ws.onopen = function() {
            console.log('Connected to chat');
            document.getElementById('messages').innerHTML += '<div>Connected!</div>';
        };
        
        ws.onmessage = function(event) {
            console.log('Message received:', event.data);
            document.getElementById('messages').innerHTML += '<div>Received: ' + event.data + '</div>';
        };
        
        function sendMessage() {
            const input = document.getElementById('messageInput');
            const message = {
                room: "general",
                body: input.value
            };
            ws.send(JSON.stringify(message));
            input.value = '';
        }
        
        document.getElementById('messageInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendMessage();
            }
        });
    </script>
</body>
</html>
```

### **Method 3: Using curl for WebSocket (Advanced)**

```bash
# This requires websocat tool
# Install: cargo install websocat
echo '{"room": "general", "body": "Hello from curl!"}' | websocat "ws://localhost:8081/ws?token=YOUR_JWT_TOKEN"
```

## 🧪 **Complete Testing Workflow**

### **Step 1: Verify Services are Running**
```bash
kubectl get pods -n go-message-app
```
All pods should show `1/1 Running`.

### **Step 2: Test Health Endpoints**
```bash
curl http://localhost:8080/health
curl http://localhost:8081/health
```

### **Step 3: Register and Login**
```bash
# Register
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"username": "alice", "password": "password123"}'

# Login and save token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username": "alice", "password": "password123"}' | \
  jq -r '.data.token')

echo "Your token: $TOKEN"
```

### **Step 4: Test WebSocket Chat**
```bash
# Using wscat
wscat -c "ws://localhost:8081/ws?token=$TOKEN"
# Then send: {"room": "general", "body": "Hello!"}
```

## 🔍 **Monitoring and Debugging**

### **Check Logs**
```bash
# Auth service logs
kubectl logs -n go-message-app deployment/auth -f

# Gateway service logs  
kubectl logs -n go-message-app deployment/gateway -f

# Persist service logs
kubectl logs -n go-message-app deployment/persist -f

# Kafka logs
kubectl logs -n go-message-app deployment/kafka -f
```

### **Check Kafka Topics**
```bash
# List topics
kubectl exec -n go-message-app deployment/kafka -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list

# Check messages in chat-in topic
kubectl exec -n go-message-app deployment/kafka -- /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic chat-in --from-beginning
```

### **Check Database**
```bash
# Connect to PostgreSQL
kubectl exec -it -n go-message-app deployment/db -- psql -U postgres -d go_message_app

# List users
# \dt (to see tables)
# SELECT * FROM users;
```

## 🚨 **Common Issues and Solutions**

### **"Unauthorized" WebSocket Error**
- Make sure your JWT token is valid and not expired (15-minute expiration)
- Re-login to get a fresh token

### **Connection Refused**
- Ensure port forwarding is active: `kubectl get pods -n go-message-app`
- Restart port forwarding if needed

### **Messages Not Persisting**
- Check persist service logs: `kubectl logs -n go-message-app deployment/persist`
- Verify Kafka is running: `kubectl logs -n go-message-app deployment/kafka`

## 🎯 **Advanced Testing Scenarios**

### **Multi-User Chat Test**
1. Register multiple users
2. Open multiple WebSocket connections with different tokens
3. Send messages from different users to the same room
4. Verify message broadcasting

### **Room Isolation Test**
1. Connect users to different rooms
2. Verify messages only go to users in the same room

### **Token Expiration Test**
1. Wait 15 minutes after login
2. Try to connect with expired token
3. Should receive "unauthorized" error

## 🎉 **Success Indicators**

✅ All health endpoints return `{"status":"healthy"}`  
✅ User registration returns success message  
✅ Login returns JWT token  
✅ WebSocket connection establishes successfully  
✅ Messages are sent and can be seen in Kafka logs  
✅ Persist service processes messages without errors  

Your microservices chat application is working perfectly! 🚀 

## 🌟 **NEW: Web Chat Interface**

### **Quick Start - Just Open Your Browser!**
1. **Open**: http://localhost:8081/chat
2. **Register/Login**: Use the built-in authentication form
3. **Start Chatting**: Real-time messaging with beautiful UI!

**That's it!** No need to manually handle JWT tokens or WebSocket connections. Everything is integrated! 🎉

## Prerequisites ✅

### 1. Database Setup (IMPORTANT!)
The database tables need to be created before testing. Run these commands:

```bash
# Create users table
kubectl exec -n go-message-app deployment/db -- psql -U postgres -d go-message-app -c "CREATE TABLE users (id SERIAL PRIMARY KEY, username TEXT UNIQUE NOT NULL, password TEXT NOT NULL, created_at TIMESTAMPTZ DEFAULT now());"

# Create messages table with UUID extension
kubectl exec -n go-message-app deployment/db -- psql -U postgres -d go-message-app -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"; CREATE TABLE messages (id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), room TEXT NOT NULL, author_id INT NOT NULL REFERENCES users(id), body TEXT NOT NULL, created_at TIMESTAMPTZ DEFAULT now());"

# Verify tables were created
kubectl exec -n go-message-app deployment/db -- psql -U postgres -d go-message-app -c "\dt"
```

### 2. Port Forwarding Setup
```bash
# Clean up any existing port forwards
pkill -f "kubectl port-forward"

# Set up port forwarding for both services
kubectl port-forward -n go-message-app service/auth 8080:8080 &
kubectl port-forward -n go-message-app service/gateway 8081:8081 &
```

## 🚀 **Recommended Testing Workflow**

### **Method 1: Web Chat Interface (Easiest!)**
1. **Open**: http://localhost:8081/chat in your browser
2. **Register**: Enter username/password and click "Register"
3. **Login**: Click "Login" to get your JWT token automatically
4. **Connect**: Click "Connect to Chat"
5. **Chat**: Type messages and press Enter!

**Features:**
- ✅ Built-in registration and login
- ✅ Automatic JWT token handling
- ✅ Real-time message display
- ✅ Room-based chat
- ✅ Beautiful, responsive UI
- ✅ Message timestamps and user identification

### **Method 2: Manual API Testing (Advanced)**

#### Step 1: Health Check Endpoints
```bash
# Check auth service health
curl -s http://localhost:8080/health
# Expected: {"status":"healthy"}

# Check gateway service health  
curl -s http://localhost:8081/health
# Expected: {"status":"healthy"}
```

#### Step 2: User Registration
```bash
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "testpass123"
  }'
```
**Expected Response:**
```json
{
  "success": true,
  "status_code": 200,
  "message": "Registration Successful! Please proceed to login to get JWT.",
  "data": {}
}
```

#### Step 3: User Login & Get JWT Token
```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser", 
    "password": "testpass123"
  }'
```
**Expected Response:**
```json
{
  "success": true,
  "status_code": 200,
  "message": "Successfully login",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

#### Step 4: Extract Token for WebSocket Testing
```bash
# Extract token and save to variable
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username": "testuser", "password": "testpass123"}' | \
  grep -o '"token":"[^"]*"' | cut -d'"' -f4)

echo "Your JWT Token: $TOKEN"
```

## WebSocket Chat Testing 💬

### Method 1: Web Interface (Recommended)
**Just go to**: http://localhost:8081/chat

### Method 2: Using wscat (Command Line)
```bash
# Install wscat if not already installed
npm install -g wscat

# Connect to WebSocket with your token
wscat -c "ws://localhost:8081/ws?token=YOUR_JWT_TOKEN_HERE"

# Send a message (after connection is established)
{"room": "general", "body": "Hello from wscat!"}
```

### Method 3: Using websocat (Advanced)
```bash
# Install websocat
brew install websocat  # macOS
# or download from: https://github.com/vi/websocat

# Connect and send message
echo '{"room": "general", "body": "Hello World!"}' | \
  websocat "ws://localhost:8081/ws?token=$TOKEN"
```

## 🎯 **Multi-User Testing**

### **Easy Multi-User Test with Web Interface:**
1. **Open multiple browser tabs/windows** to http://localhost:8081/chat
2. **Register different users** in each tab (user1, user2, user3, etc.)
3. **Login each user** and connect to chat
4. **Join the same room** (e.g., "general") in all tabs
5. **Send messages** from different users
6. **Watch real-time messaging** across all tabs! 🎉

### **Room Isolation Test:**
1. Connect users to different rooms ("room1", "room2")
2. Send messages in each room
3. Verify messages only appear in the correct room

## Complete Testing Workflow 🔄

### Quick Test Script
```bash
#!/bin/bash

echo "🚀 Testing Go Message App..."

# 1. Health checks
echo "1. Checking service health..."
curl -s http://localhost:8080/health && echo
curl -s http://localhost:8081/health && echo

# 2. Test chat interface
echo "2. Testing chat interface..."
curl -s http://localhost:8081/chat | head -5

echo "3. 🌟 Open your browser to: http://localhost:8081/chat"
echo "4. Register, login, and start chatting!"
```

## Monitoring and Debugging 🔍

### Check Service Logs
```bash
# Auth service logs
kubectl logs -n go-message-app deployment/auth -f

# Gateway service logs  
kubectl logs -n go-message-app deployment/gateway -f

# Persist service logs
kubectl logs -n go-message-app deployment/persist -f

# Database logs
kubectl logs -n go-message-app deployment/db -f

# Kafka logs
kubectl logs -n go-message-app deployment/kafka -f
```

### Check Kafka Topics
```bash
# List Kafka topics
kubectl exec -n go-message-app deployment/kafka -- \
  /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list

# Monitor chat-in topic
kubectl exec -n go-message-app deployment/kafka -- \
  /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic chat-in \
  --from-beginning
```

### Check Database
```bash
# Connect to PostgreSQL
kubectl exec -it -n go-message-app deployment/db -- \
  psql -U postgres -d go-message-app

# Check users
SELECT * FROM users;

# Check messages  
SELECT * FROM messages;

# Exit with \q
```

## Common Issues and Solutions 🛠️

### Issue: "relation 'users' does not exist"
**Solution:** Run the database setup commands from Prerequisites section.

### Issue: Chat interface not loading
**Solution:** 
- Verify port forwarding: `kubectl port-forward -n go-message-app service/gateway 8081:8081 &`
- Check gateway pod: `kubectl get pods -n go-message-app`

### Issue: "Unauthorized" WebSocket Error
**Solution:** 
- Use the web interface for automatic token handling
- Ensure JWT token is valid and not expired (15-minute TTL)

### Issue: "Connection refused"
**Solution:**
- Verify port forwarding is active: `ps aux | grep "kubectl port-forward"`
- Restart port forwarding if needed

### Issue: Messages not persisting
**Solution:**
- Check persist service logs: `kubectl logs -n go-message-app deployment/persist -f`
- Restart persist service: `kubectl rollout restart deployment/persist -n go-message-app`

## Advanced Testing Scenarios 🎯

### Multi-User Chat Test
1. Open multiple browser tabs with the chat interface
2. Register different users in each tab
3. Connect all users to the same room
4. Send messages from different users
5. Verify all users receive messages in real-time

### Room Isolation Test
1. Connect two users to different rooms ("room1" and "room2")
2. Send messages in each room
3. Verify messages only appear in the correct room

### Token Expiration Test
1. Wait 15+ minutes after login
2. Try to send a message with expired token
3. Should receive connection error
4. Login again to get fresh token

## Success Indicators ✅

Your microservices chat application is working correctly when:

- ✅ Chat interface loads at http://localhost:8081/chat
- ✅ User registration and login work in the web interface
- ✅ WebSocket connections establish automatically
- ✅ Messages sent via web interface appear in real-time
- ✅ Multiple users can chat simultaneously
- ✅ Room isolation works correctly
- ✅ All Kubernetes pods are in "Running" state
- ✅ Health endpoints return `{"status":"healthy"}`

## Quick Commands Reference 📋

```bash
# Open the chat interface
open http://localhost:8081/chat  # macOS
# or just open the URL in your browser

# Check all pods status
kubectl get pods -n go-message-app

# Restart port forwarding if needed
kubectl port-forward -n go-message-app service/auth 8080:8080 &
kubectl port-forward -n go-message-app service/gateway 8081:8081 &

# Test health endpoints
curl -s http://localhost:8080/health && curl -s http://localhost:8081/health
```

---

🎉 **Congratulations!** Your Go microservices chat application now has a beautiful web interface at **http://localhost:8081/chat**! 