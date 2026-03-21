FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -o /pumice

FROM alpine:3.21

COPY --from=builder /pumice /usr/local/bin/pumice
COPY --chmod=755 entrypoint.sh /

WORKDIR /site

ENTRYPOINT [ "/entrypoint.sh" ]
