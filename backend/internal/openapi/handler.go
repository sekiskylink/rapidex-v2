package openapi

import (
	"fmt"
	"net/http"
	"sync"

	"basepro/backend/internal/openapi/generated"
	"github.com/gin-gonic/gin"
	"go.yaml.in/yaml/v3"
)

const scalarHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>SukumadPro API Docs</title>
    <style>
      html, body { height: 100%%; margin: 0; }
      body { background: #0b1020; }
    </style>
  </head>
  <body>
    <script id="api-reference" data-url="/openapi.yaml"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>
`

var (
	specYAMLOnce sync.Once
	specYAML     []byte
	specYAMLErr  error
)

func specBytes() ([]byte, error) {
	specYAMLOnce.Do(func() {
		swagger, err := generated.GetSwagger()
		if err != nil {
			specYAMLErr = fmt.Errorf("load embedded openapi spec: %w", err)
			return
		}
		specYAML, specYAMLErr = yaml.Marshal(swagger)
		if specYAMLErr != nil {
			specYAMLErr = fmt.Errorf("marshal openapi spec as yaml: %w", specYAMLErr)
		}
	})
	return specYAML, specYAMLErr
}

func RegisterRoutes(r *gin.Engine) {
	r.GET("/openapi.yaml", serveSpec)
	r.GET("/docs", serveDocs)
}

func serveSpec(c *gin.Context) {
	payload, err := specBytes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "OPENAPI_SPEC_UNAVAILABLE",
				"message": "openapi spec is unavailable",
			},
		})
		return
	}

	c.Data(http.StatusOK, "application/yaml; charset=utf-8", payload)
}

func serveDocs(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(scalarHTML))
}
