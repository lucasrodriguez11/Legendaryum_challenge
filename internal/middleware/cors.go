package middleware

import (
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func CORSMiddleware() fiber.Handler {
	// Verificar si estamos en modo desarrollo
	isDevelopment := os.Getenv("ENV") == "development" ||
		os.Getenv("NODE_ENV") == "development" ||
		os.Getenv("GO_ENV") == "development"

	// Obtener orígenes específicos del .env si existen
	envOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if envOrigins != "" && envOrigins != "*" {
		allowedOrigins = strings.Split(envOrigins, ",")
		// Limpiar espacios en blanco
		for i, origin := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(origin)
		}
	}

	config := cors.Config{
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH,HEAD",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-HTTP-Method-Override",
		ExposeHeaders:    "Content-Length, Content-Type, Authorization",
		AllowCredentials: true,
		MaxAge:           86400, // 24 horas
	}

	// Si CORS_ALLOWED_ORIGINS es "*" o estamos en desarrollo, usar función permisiva
	if envOrigins == "*" || isDevelopment {
		config.AllowOriginsFunc = func(origin string) bool {
			return isAllowedOrigin(origin, allowedOrigins, isDevelopment)
		}
	} else if len(allowedOrigins) > 0 {
		// En producción con orígenes específicos
		config.AllowOrigins = strings.Join(allowedOrigins, ",")
	} else {
		// Fallback restrictivo para producción
		config.AllowOriginsFunc = func(origin string) bool {
			return isProductionOrigin(origin)
		}
	}

	return cors.New(config)
}

// isAllowedOrigin verifica si un origen está permitido (versión mejorada)
func isAllowedOrigin(origin string, explicitOrigins []string, isDevelopment bool) bool {
	// Permitir orígenes vacíos (requests sin CORS)
	if origin == "" {
		return true
	}

	// Verificar orígenes explícitamente permitidos
	for _, allowed := range explicitOrigins {
		if allowed != "" && origin == allowed {
			return true
		}
	}

	// En desarrollo, permitir patrones más amplios
	if isDevelopment {
		return isDevelopmentOrigin(origin)
	}

	return false
}

// isDevelopmentOrigin verifica patrones permitidos en desarrollo (más permisivo)
func isDevelopmentOrigin(origin string) bool {
	developmentPatterns := []*regexp.Regexp{
		// Localhost en cualquier puerto
		regexp.MustCompile(`^https?://localhost(:\d+)?$`),
		regexp.MustCompile(`^https?://127\.0\.0\.1(:\d+)?$`),
		regexp.MustCompile(`^https?://0\.0\.0\.0(:\d+)?$`),

		// Redes privadas RFC 1918
		regexp.MustCompile(`^https?://10\.\d+\.\d+\.\d+(:\d+)?$`),                 // 10.0.0.0/8
		regexp.MustCompile(`^https?://172\.(1[6-9]|2\d|3[01])\.\d+\.\d+(:\d+)?$`), // 172.16.0.0/12
		regexp.MustCompile(`^https?://192\.168\.\d+\.\d+(:\d+)?$`),                // 192.168.0.0/16

		// Docker networks específicos
		regexp.MustCompile(`^https?://172\.17\.\d+\.\d+(:\d+)?$`), // Docker default bridge
		regexp.MustCompile(`^https?://172\.18\.\d+\.\d+(:\d+)?$`), // Docker custom networks
		regexp.MustCompile(`^https?://172\.19\.\d+\.\d+(:\d+)?$`),
		regexp.MustCompile(`^https?://172\.2[0-9]\.\d+\.\d+(:\d+)?$`),

		// Link-local addresses
		regexp.MustCompile(`^https?://169\.254\.\d+\.\d+(:\d+)?$`), // 169.254.0.0/16

		// Docker Machine / VirtualBox (común en Windows/Mac con Docker Toolbox)
		regexp.MustCompile(`^https?://192\.168\.99\.\d+(:\d+)?$`), // Red por defecto de Docker Machine

		// Permitir cualquier IP local de desarrollo (más permisivo)
		regexp.MustCompile(`^https?://[\d.]+:\d+$`), // Cualquier IP:puerto
	}

	// Verificar patrones
	for _, pattern := range developmentPatterns {
		if pattern.MatchString(origin) {
			return true
		}
	}

	return false
}

// isProductionOrigin verifica orígenes para producción (más restrictivo)
func isProductionOrigin(origin string) bool {
	if origin == "" {
		return true
	}

	// En producción, solo permitir HTTPS (excepto localhost para testing)
	httpsPattern := regexp.MustCompile(`^https://`)
	localhostPattern := regexp.MustCompile(`^https?://localhost(:\d+)?$`)

	return httpsPattern.MatchString(origin) || localhostPattern.MatchString(origin)
}

// GetLocalIPs obtiene las IPs locales de la máquina (función helper)
func GetLocalIPs() []string {
	var ips []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip != nil {
				ips = append(ips, ip.String())
			}
		}
	}

	return ips
}
