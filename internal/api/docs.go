package api

import (
	_ "embed"
	"net/http"
)

// openapiSpec — контракт API, встроен в бинарник (go:embed),
// поэтому рядом с сервисом не нужно класть никаких файлов.
//
//go:embed openapi.yaml
var openapiSpec []byte

// docsPage — Swagger UI. Скрипты и стили грузит браузер с CDN,
// сам сервис отдаёт только эту страницу и openapi.yaml.
const docsPage = `<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>News API</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({ url: "/api/docs/openapi.yaml", dom_id: "#swagger-ui" });
  </script>
</body>
</html>`

// docs — GET /api/docs — Swagger UI для фронтендеров.
func docs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(docsPage))
}

// spec — GET /api/docs/openapi.yaml — сама спецификация.
func spec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(openapiSpec)
}
