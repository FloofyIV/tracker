FROM golang:1.19-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tracker

FROM alpine:3
WORKDIR /app
COPY --from=builder /app/tracker /app/tracker

ENV PLACE=""
ENV WEBHOOK=""
ENV ROLE=""

ENTRYPOINT ["/app/tracker"]