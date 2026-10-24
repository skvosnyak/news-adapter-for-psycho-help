# --- сборка ---
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/news-service .

# --- итоговый образ: только бинарник и сертификаты для HTTPS к VK ---
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/news-service /news-service
USER 65534
EXPOSE 8080
ENTRYPOINT ["/news-service"]
