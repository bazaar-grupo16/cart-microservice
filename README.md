# Bazaar — Microservicio de Carrito

Este es el microservicio de **Carrito** del ecosistema **Bazaar**, encargado de gestionar los carritos de compras de los usuarios de forma persistente.

## Stack Tecnológico

| Capa | Tecnología |
|------|-----------|
| Lenguaje | Golang 1.25.0+ |
| Framework Web | Gin Gonic |
| Base de Datos | MongoDB 8.0 |
| Logging | Zap Logger (Uber) |
| Contenedor | Docker (multi-stage) |
| Orquestación | Kubernetes (DOKS) |
| CI/CD | GitHub Actions |

## Arquitectura

El microservicio sigue una arquitectura por capas para facilitar el mantenimiento y testing:

- **Controller**: Gestiona los endpoints HTTP, valida los parámetros de entrada y formatea las respuestas (utiliza Problem JSON para errores).
- **Service**: Contiene la lógica de negocio. Se comunica con el repositorio y coordina llamadas externas (ej. al catálogo).
- **Repository**: Capa de persistencia que encapsula todas las operaciones de MongoDB utilizando el driver oficial (`mongo-driver/v2`).
- **Models**: Definiciones de estructuras de datos (Cart, CartItem, Price) compartidas entre capas.
- **Middleware**: Manejo de concerns transversales como validación de Content-Type y recuperación de pánicos.

### Detalles de Implementación

- **Snapshot de Productos**: Al agregar un ítem al carrito, se guarda un "snapshot" del título y el precio unitario del catálogo. Esto garantiza que el precio se mantenga estable durante la sesión de compra.
- **Precisión de Precios**: Se utiliza un sistema de alta precisión (`math/big`) para evitar errores de redondeo de punto flotante en los cálculos del total.
- **Retry Logic**: El servicio implementa mecanismos de reintento y validación de disponibilidad para conexiones a la base de datos y servicios externos.

## Estructura del Proyecto

```
├── src/
│   ├── config/         # Configuración y conexión a DB
│   ├── controller/     # Handlers de la API (Gin)
│   ├── middleware/     # Middlewares de Gin
│   ├── models/         # Estructuras de datos/entidades
│   ├── repository/     # Operaciones de MongoDB
│   ├── router/         # Definición de rutas
│   ├── service/        # Lógica de negocio y clientes externos
│   ├── utils/          # Utilidades (JSON helpers, etc.)
│   └── main.go         # Punto de entrada
├── scripts/
│   └── run_tests.sh    # Script para ejecutar tests con Docker
├── Dockerfile          # Imagen de producción
├── docker-compose.yaml # Entorno de desarrollo local
├── go.mod              # Dependencias de Go
└── .env.example        # Template de variables de entorno
```

## Configuración Local

### Requisitos
- [Go 1.25+](https://go.dev/doc/install)
- [Docker](https://docs.docker.com/get-docker/) y [Docker Compose](https://docs.docker.com/compose/install/)

### Pasos para levantar el servicio
1. **Configurar el entorno**:
   ```bash
   cp .env.example .env
   # Ajustar variables si es necesario (MONGO_URI, PORT, etc.)
   ```

2. **Levantar con Docker Compose**:
   ```bash
   docker compose up --build
   ```
   Esto levantará una instancia de MongoDB y la API en el puerto `8080` (por defecto).

3. **Ejecutar localmente (para desarrollo)**:
   ```bash
   cd src
   go run main.go
   ```

## API Endpoints

### Salud (Probes)
- `GET /livez`: Liveness probe — responde 200 OK.
- `GET /readyz`: Readiness probe — verifica conectividad con MongoDB.

### Carrito
| Método | Ruta | Descripción |
|--------|------|-------------|
| `GET` | `/cart/:user_id` | Obtiene el carrito del usuario. |
| `DELETE` | `/cart/:user_id` | Vacía el carrito del usuario. |
| `POST` | `/cart/:user_id/items` | Agrega un producto al carrito (acepta `product_id`). |
| `DELETE` | `/cart/:user_id/items/:product_id` | Elimina un producto específico del carrito. |

## Tests

El proyecto incluye tests de integración que corren contra una instancia real de MongoDB en Docker.

```bash
# Ejecutar todos los tests
./run_tests.sh
```

## Docker & CI/CD

- **Dockerfile**: Utiliza un build multi-stage para generar una imagen optimizada basada en `alpine`.
- **CI/CD**: Ubicado en `.github/workflows/`, ejecuta tests automáticamente en cada Pull Request y despliega a Kubernetes en cada push a `main`.
