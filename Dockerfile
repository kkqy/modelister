# --- 阶段 1：构建前端（Vite -> internal/webui/dist） ---
FROM node:22-bookworm AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci || npm install
COPY frontend/ ./
# vite.config.js 输出到 ../internal/webui/dist，即 /src/internal/webui/dist。
RUN npm run build

# --- 阶段 2：构建嵌入前端资源的 Go 二进制 ---
FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 拷入刚构建出的前端资源，供 //go:embed 嵌入。
COPY --from=web /src/internal/webui/dist ./internal/webui/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/modelister ./cmd/modelister

# --- 阶段 3：运行时镜像 ---
FROM debian:bookworm-slim

WORKDIR /app
RUN mkdir -p /data
COPY --from=build /out/modelister /app/modelister
EXPOSE 8080
ENV APP_DATABASE_PATH=/data/modelister.db
ENV APP_HTTP_ADDR=:8080
CMD ["/app/modelister"]
