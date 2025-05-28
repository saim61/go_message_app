# Real time Go Messaging Application
Event Driven, Real Time Messaging app in Go

## Tools and Technologies
- Go
- PostGreSQL
- Apache Kafka
- golang-migrate
- Docker
- Kubernetes

## Setting up the project
- Download and install golang-migrate
- Download PostGreSQL and Go. Assuming you already have this installed but if not, please install it. Its not that hard to do so.

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


Install the tools and technologies and start the server
```sh
go run cmd/auth/main.go
```

Create a user and login
```
curl -X POST localhost:8080/register -d '{"username":"bobthebuilder","password":"iamabuilder"}' -H 'Content-Type: application/json'
curl -X POST localhost:8080/login -d '{"username":"bobthebuilder","password":"iamabuilder"}' -H 'Content-Type: application/json'
```