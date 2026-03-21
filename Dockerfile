FROM golang:1.24.2-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -o /pumice

FROM scratch

COPY --from=builder /pumice /usr/local/bin/pumice

WORKDIR /site

ENTRYPOINT [ "pumice" ]
