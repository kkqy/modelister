FROM golang:1.22-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/modelister ./cmd/modelister

FROM debian:bookworm-slim

WORKDIR /app
RUN mkdir -p /data
COPY --from=build /out/modelister /app/modelister
EXPOSE 8080
ENV APP_DATABASE_PATH=/data/modelister.db
ENV APP_HTTP_ADDR=:8080
CMD ["/app/modelister"]
