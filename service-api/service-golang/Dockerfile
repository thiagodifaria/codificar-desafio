FROM golang:1.26-alpine AS build

WORKDIR /src
COPY service-api/service-golang/go.mod service-api/service-golang/go.sum ./
RUN go mod download

COPY service-api/service-golang/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /api ./cmd/api

FROM alpine:3.23

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

COPY --from=build /api /usr/local/bin/api
USER app
EXPOSE 8080
ENTRYPOINT ["api"]
