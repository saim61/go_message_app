# syntax=docker/dockerfile:1
ARG SERVICE
FROM golang:1.22-alpine AS builder

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
