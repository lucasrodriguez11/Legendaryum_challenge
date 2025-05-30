# Legendaryum Backend Challenge

Este proyecto es una API RESTful de gestión de tareas desarrollada en Go.

## Características Implementadas

- **Autenticación de Usuarios:** Registro y login de usuarios con JWT.
- **Gestión de Tareas:** CRUD completo para tareas.
- **Filtrado de Tareas:** Permite filtrar tareas por estado y prioridad.
- **Autorización:** Endpoints de tareas protegidos con autenticación JWT.
- **Base de Datos:** Integración con PostgreSQL usando GORM.
- **Migraciones:** Gestión de esquema de base de datos con `golang-migrate`.
- **Hashing de Contraseñas:** Uso seguro de bcrypt.
- **Dockerización:** Contenedorización de la aplicación y la base de datos con Docker Compose.
- **Documentación API:** Documentación interactiva con Swagger.
- **Pruebas:** Cobertura de tests unitarios para funcionalidades.

## Stack Tecnológico

- **Lenguaje:** Go (1.24+)
- **Framework Web:** Fiber (v2.x)
- **ORM:** GORM (v1.x)
- **Base de Datos:** PostgreSQL (15+)
- **Migraciones DB:** golang-migrate (v4.x)
- **JWT:** golang-jwt/jwt/v5 (v5.x)
- **Hashing:** golang.org/x/crypto/bcrypt
- **Contenedores:** Docker & Docker Compose
- **Documentación API:** Swaggo (fiber-swagger v1.x, swag v1.x)
- **Testing:** Testify (v1.x)

## Configuración y Ejecución

## Requisitos Previos

- Docker y Docker Compose instalados
- Go 1.24+ (para desarrollo local)
- PostgreSQL 15+ (para desarrollo local sin Docker)
- Git

1.  **Clona el repositorio:**
    ```bash
    git clone https://github.com/lucasrodriguez11/Legendaryum_challenge.git
    ```

2.  **Configura las variables de entorno:**
    Copia el archivo de ejemplo (.env.example) y actualiza las variables según tu entorno (especialmente las de base de datos) e IP de VM.

3.  **Construye y levanta los servicios con Docker Compose:**
    Este comando construirá la imagen de Docker para la API, descargará la imagen de PostgreSQL y levantará ambos servicios. Las migraciones de base de datos se aplicarán automáticamente al iniciar el contenedor de la API.
    ```bash
    docker-compose up --build -d
    ```
    *   `--build`: Reconstruye la imagen de la API.
    *   `-d`: Ejecuta los servicios en segundo plano (detached mode).

4.  **Verifica que los servicios estén corriendo:**
    ```bash
    docker-compose ps
    ```
    Deberías ver los contenedores `legendaryum-api-1` y `legendaryum-db-1` en estado `Up`.

## Base de Datos y Migraciones

La base de datos PostgreSQL se inicia como un servicio de Docker Compose o bien de manera local se debe crear una base de datos con un gestor para poder probar la app localmente sin docker. Las migraciones definidas en el directorio `migrations/` se ejecutan automáticamente cada vez que el contenedor `api` se inicia (`database.RunMigrations(cfg)` en `cmd/api/main.go`) o cuando se inicia la app de forma local.

## Tests (Solo Local (se debe crear la bd))

Los tests unitarios se encuentran en el directorio `tests/`. Puedes ejecutarlos con el siguiente comando desde la raíz del proyecto:

```bash
go test -v ./tests/...
```

Esto ejecutará todos los tests dentro del directorio `tests/`.

## Documentación API (Swagger)

La documentación interactiva de la API está disponible a través de Swagger UI.

1.  **Accede a la documentación:**
    Una vez que el servidor de la API esté corriendo (ya sea localmente con `go run` o vía Docker Compose), abre tu navegador y navega a:
    ```
    http://localhost:8080/docs/ 
    ```

## Endpoints de la API

Resumen de los endpoints principales:

-   **`POST /auth/register`**: Registra un nuevo usuario.
-   **`POST /auth/login`**: Inicia sesión y devuelve un token JWT.

-   **`GET /tasks`**: Lista tareas (creadas por o asignadas al usuario autenticado). Soporta filtrado por `status` y `priority` (query params).
-   **`POST /tasks`**: Crea una nueva tarea; Json ejemplo:
    {
         "title": "Implementar autenticación JWT y refresh tokens",
         "description": "Desarrollar el sistema de autenticación usando JWT.",
         "due_date": "2024-06-20T00:00:00Z", // ISO 8601 - YYYY-MM-DDThh:mm:ssZ
          "priority": "high",
         "status": "in_progress",
         "assignee_id": "UUID de User" // Opcional, para reasignar la tarea
    }
-   **`GET /tasks/{id}`**: Obtiene detalles de una tarea específica por ID.
-   **`PUT /tasks/{id}`**: Actualiza una tarea específica por ID; Json Ejemplo:
    {
        "assignee_id": "UUID de User"
    }
-   **`DELETE /tasks/{id}`**: Elimina una tarea específica por ID.

---
Desarrollado por:
Lucas Nahuel Rodriguez