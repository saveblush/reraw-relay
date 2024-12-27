# Define version
ARG GO_VERSION=1.23.1
FROM golang:${GO_VERSION}-bullseye AS base

 
# Build source code
FROM base AS builder

## Create user
RUN adduser \
  --disabled-password \
  --gecos "" \
  --home "/nonexistent" \
  --shell "/sbin/nologin" \
  --no-create-home \
  --uid 65532 \
  gouser

## Change ownership
RUN mkdir /app
RUN chown gouser:gouser /app

## Set working directory
WORKDIR /app

## Copy dependency
COPY go.mod go.sum ./

## Get all dependencies
RUN go mod download
RUN go mod verify

## Copy the source code
COPY . .

## Build app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o main .


# Production
FROM scratch AS runner
WORKDIR /app

## Copy os bundle
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

USER gouser

## Copy app
COPY --from=builder /app/main .
COPY --from=builder --chown=gouser:gouser /app/configs ./configs

ENV TZ=Asia/Bangkok

EXPOSE 8070

CMD ["./main"]