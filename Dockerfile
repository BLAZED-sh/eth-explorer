# --- frontend ---
FROM --platform=$BUILDPLATFORM node:20-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# --- backend ---
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist ./web/dist
RUN mkdir -p /data
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w" -o /blazed-explorer ./cmd/server

# --- runtime ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /blazed-explorer /blazed-explorer
COPY --from=build --chown=65532:65532 /data /data
ENV DB_PATH=/data/explorer.db
WORKDIR /data
EXPOSE 8080
ENTRYPOINT ["/blazed-explorer"]
