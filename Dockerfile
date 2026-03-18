FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath -ldflags="-s -w" \
    -o gospeed-server ./cmd/gospeed-server

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/gospeed-server /gospeed-server

EXPOSE 9000

ENTRYPOINT ["/gospeed-server"]
