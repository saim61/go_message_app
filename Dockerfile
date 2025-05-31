# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS builder
ARG SERVICE
ENV GOTOOLCHAIN=auto
RUN echo "this is the service: $SERVICE" 

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /bin/service ./cmd/${SERVICE}

FROM gcr.io/distroless/static:nonroot
ARG SERVICE
USER 65532:65532
COPY --from=builder /bin/service /service
ENTRYPOINT ["/service"]
