# Real time Go Messaging Application
Event Driven, Real Time Messaging app in Go

## Tools and Technologies
- Go
- PostGreSQL
- Apache Kafka
- golang-migrate
- Docker
- Kubernetes
- Web Sockets

## Setting up the project
- Download and install golang-migrate
- Download `PostGreSQL` and `Go`. Assuming you already have this installed but if not, please install it.
- Download `Node JS` which gives you both `node` and `npm`
- Download `wscat`

## Create your Database
Run the following command in your terminal (make sure PostGreSQL is working)
```sh
psql -h localhost -U your_username -w -c "create database go_message_app;"

// saving the PostGreSQL URL in a variable for simplicity
export POSTGRESQL_URL='postgresql://user:password@localhost:5432/go_message_app?sslmode=disable'
```

## Run the migrations
```sh
cd migrations
migrate -database "$POSTGRESQL_URL" -path . up

```

## Running the server
Install the tools and technologies and start the server
```sh
go run cmd/auth/main.go
```

## Kafka Docker Image
Run Kafka Docker Compose file using this command (runs on port 9092 so make sure its free for use)
```sh
docker compose -f docker-compose.kafka.yml up -d
```

## To shut down Kafka
```sh
docker compose -f docker-compose.kafka.yml down
```

## To view the messages
- You can either use `kcat` tool on your mac by using the following commands
```sh
brew install kcat
kcat -b localhost:9092 -t chat-in -C -J
```

- From inside the docker container you can do this (a much cleaner view):
```sh
docker exec -it kafka bash

# when inside the container, run this:
/opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic chat-in \
  --from-beginning
```

Create a user and login
```
curl -X POST localhost:8080/register -d '{"username":"bobthebuilder","password":"iamabuilder"}' -H 'Content-Type: application/json'
curl -X POST localhost:8080/login -d '{"username":"bobthebuilder","password":"iamabuilder"}' -H 'Content-Type: application/json'
```

Docker build commands (run each command separately):
```sh
docker build --build-arg SERVICE=auth -t go-messge-app-auth:latest .
docker build --build-arg SERVICE=gateway -t go-messge-app-gateway:latest .
docker build --build-arg SERVICE=persist -t go-messge-app-persist:latest .
```

Kubectl apply commands (run each separately):
```sh
kubectl apply -f 30-auth.yaml
kubectl apply -f 31-gateway.yaml
kubectl apply -f 32-persist.yaml
```

Start a local registry:
```sh
docker run -d --restart=always -p 5000:5000 --name registry registry:2
```

### One-time local setup

```bash
# 1. start a tiny registry that the cluster can pull from
docker run -d --restart=always -p 5000:5000 --name registry registry:2

# 2. build + push images
docker build --build-arg SERVICE=auth    -t host.docker.internal:5000/go-message-app/auth:latest    . && docker push host.docker.internal:5000/go-message-app/auth:latest
docker build --build-arg SERVICE=gateway -t host.docker.internal:5000/go-message-app/gateway:latest . && docker push host.docker.internal:5000/go-message-app/gateway:latest
docker build --build-arg SERVICE=persist -t host.docker.internal:5000/go-message-app/persist:latest . && docker push host.docker.internal:5000/go-message-app/persist:latest

# 3. deploy to Docker-Desktop Kubernetes
kubectl apply -f k8s/
kubectl -n go-message-app get pods   # wait until all are Running
```

DELETE EVERYTHING
```sh
kubectl delete namespace go-message-app --wait=true 2>/dev/null || true
docker rm -f registry 2>/dev/null || true
docker image rm -f \
  go-message-app-auth:latest \
  go-message-app-gateway:latest \
  go-message-app-persist:latest \
  localhost:5000/go-message-app/auth:latest \
  localhost:5000/go-message-app/gateway:latest \
  localhost:5000/go-message-app/persist:latest 2>/dev/null || true \
  host.docker.internal/go-message-app/auth:latest \
  host.docker.internal/go-message-app/gateway:latest \
  host.docker.internal/go-message-app/persist:latest 2>/dev/null || true
  host.docker.internal:5000/go-message-app/auth:latest \
  host.docker.internal:5000/go-message-app/gateway:latest \
  host.docker.internal:5000/go-message-app/persist:latest 2>/dev/null || true
```

## TODO
- error checks and validations
- test cases