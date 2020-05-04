# Use base golang image from Docker Hub
FROM golang:1.12-alpine AS build
RUN apk add --update --no-cache git
WORKDIR /src/aws-secrets-manager
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go build -o /app -v ./cmd/aws-secrets-manager

FROM alpine
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=build /app /.
ENTRYPOINT ["/app"]