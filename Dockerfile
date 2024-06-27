FROM golang:1.22-alpine as builder

WORKDIR /app
COPY src .
RUN go mod download 
RUN go build -a -installsuffix meraki_exporter -o app .

# RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .



FROM alpine:latest as prod

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app .
EXPOSE 8080
CMD ["./app"]