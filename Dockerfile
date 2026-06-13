# --- build stage ---
FROM golang:1.24-alpine AS build
WORKDIR /src

# Stdlib-only project: go.mod is enough, there are no modules to download.
COPY go.mod ./
COPY . .

# Static, stripped binary so it runs on a scratch/distroless base.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# --- runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot

# TLS roots so the binary can call api.anthropic.com over HTTPS.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /server /server

# Provide ANTHROPIC_API_KEY at runtime; PORT defaults to 8080.
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]
