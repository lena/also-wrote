# Stage 1: build frontend
FROM node:20-alpine AS frontend
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: build and run Go server
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY internal/ ./internal/
COPY templates/ ./templates/
COPY static/ ./static/
COPY --from=frontend /app/dist ./frontend/dist
RUN go build -o server .

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/server ./
COPY --from=builder /app/frontend ./frontend
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
ENV PORT=8080
EXPOSE 8080
CMD ["./server"]
