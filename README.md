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


## TODO
- error checks and validations
- error responses
- test cases
- code cleanup and refactor
- logging in gateway/main.go, auth/main.go, persist/main.go 