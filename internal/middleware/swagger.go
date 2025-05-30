package middleware

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

//go:embed swagger.json
var swaggerDocs embed.FS

// SwaggerUI configura el middleware para servir la documentación Swagger
func SwaggerUI() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Configurar headers CORS para todas las rutas de Swagger
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")

		// Manejar preflight requests
		if c.Method() == "OPTIONS" {
			return c.SendStatus(fiber.StatusOK)
		}

		// Servir el archivo swagger.json dinámicamente
		if c.Path() == "/swagger.json" {
			swaggerFile, err := swaggerDocs.ReadFile("swagger.json")
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Error al leer la documentación Swagger",
				})
			}

			// Obtener la URL base correcta
			baseURL := getBaseURL(c)

			// Modificar el JSON para usar la URL correcta
			swaggerContent := string(swaggerFile)
			swaggerContent = strings.ReplaceAll(swaggerContent, `"host": "localhost:8080"`, fmt.Sprintf(`"host": "%s"`, getHostFromURL(baseURL)))

			c.Set("Content-Type", "application/json")
			return c.SendString(swaggerContent)
		}

		// Servir la interfaz de Swagger UI
		if c.Path() == "/docs" || c.Path() == "/docs/" {
			// Obtener la URL base del servidor
			baseURL := getBaseURL(c)

			html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Legendaryum API - Documentación</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css">
    <style>
        body {
            margin: 0;
            padding: 0;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
        }
        #swagger-ui {
            max-width: 1460px;
            margin: 0 auto;
        }
        .topbar {
            background-color: #1f2937;
            padding: 10px 0;
        }
        .topbar-wrapper {
            max-width: 1460px;
            margin: 0 auto;
            padding: 0 20px;
        }
        .topbar-wrapper .link {
            color: #ffffff;
            font-size: 1.5em;
            font-weight: bold;
            text-decoration: none;
        }
        .info-banner {
            background-color: #f3f4f6;
            border-left: 4px solid #3b82f6;
            padding: 10px 20px;
            margin: 20px;
            border-radius: 4px;
        }
        .info-banner code {
            background-color: #e5e7eb;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: monospace;
        }
    </style>
</head>
<body>
    <div class="topbar">
        <div class="topbar-wrapper">
            <a href="/docs" class="link">Legendaryum API</a>
        </div>
    </div>
    <div class="info-banner">
        <strong>Base URL:</strong> <code id="base-url-display">%s</code>
        <br><small>Todas las peticiones se realizarán a esta URL base</small>
    </div>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-standalone-preset.js"></script>
    <script>
        const BASE_URL = '%s';
        
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: BASE_URL + "/swagger.json",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                docExpansion: "list",
                defaultModelsExpandDepth: 1,
                defaultModelExpandDepth: 1,
                defaultModelRendering: "model",
                displayRequestDuration: true,
                filter: true,
                showExtensions: true,
                showCommonExtensions: true,
                supportedSubmitMethods: [
                    "get", "put", "post", "delete", "options", "head", "patch", "trace"
                ],
                tryItOutEnabled: true,
                requestInterceptor: function(request) {
                    console.log('Request interceptor - Original URL:', request.url);
                    
                    // Si la URL no tiene protocolo, agregarle la base URL
                    if (request.url && !request.url.startsWith('http')) {
                        if (request.url.startsWith('/')) {
                            request.url = BASE_URL + request.url;
                        } else {
                            request.url = BASE_URL + '/' + request.url;
                        }
                    }
                    
                    console.log('Request interceptor - Final URL:', request.url);
                    return request;
                },
                responseInterceptor: function(response) {
                    console.log('Response interceptor:', response);
                    return response;
                },
                onComplete: function() {
                    console.log('Swagger UI loaded with base URL:', BASE_URL);
                }
            });

            window.ui = ui;
            
            // Actualizar el display de URL base
            document.getElementById('base-url-display').textContent = BASE_URL;
        }
    </script>
</body>
</html>`, baseURL, baseURL)

			c.Set("Content-Type", "text/html; charset=utf-8")
			return c.SendString(html)
		}

		return c.Next()
	}
}

// getBaseURL obtiene la URL base del servidor - VERSIÓN MEJORADA
func getBaseURL(c *fiber.Ctx) string {
	// Obtener el esquema
	scheme := "http"
	if c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	var host string

	// PRIORIDAD DE RESOLUCIÓN:
	// 1. PUBLIC_HOST desde .env (máxima prioridad para Docker/VM)
	// 2. X-Forwarded-Host (proxy/load balancer)
	// 3. Host header (lo que envía el cliente)
	// 4. Fallback a localhost con puerto

	if publicHost := os.Getenv("PUBLIC_HOST"); publicHost != "" {
		host = publicHost
		fmt.Printf("Swagger getBaseURL: Usando PUBLIC_HOST: %s\n", host)
	} else if fwdHost := c.Get("X-Forwarded-Host"); fwdHost != "" {
		host = fwdHost
		fmt.Printf("Swagger getBaseURL: Usando X-Forwarded-Host: %s\n", host)
	} else if hostHeader := c.Get("Host"); hostHeader != "" {
		host = hostHeader
		fmt.Printf("Swagger getBaseURL: Usando Host header: %s\n", host)
	} else {
		// Fallback final
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		host = fmt.Sprintf("localhost:%s", port)
		fmt.Printf("Swagger getBaseURL: Fallback a localhost: %s\n", host)
	}

	finalURL := fmt.Sprintf("%s://%s", scheme, host)
	fmt.Printf("Swagger getBaseURL: URL final generada: %s\n", finalURL)
	return finalURL
}

// getHostFromURL extrae solo el host de una URL completa
func getHostFromURL(fullURL string) string {
	// Remover el esquema
	if strings.HasPrefix(fullURL, "https://") {
		return strings.TrimPrefix(fullURL, "https://")
	}
	if strings.HasPrefix(fullURL, "http://") {
		return strings.TrimPrefix(fullURL, "http://")
	}
	return fullURL
}
