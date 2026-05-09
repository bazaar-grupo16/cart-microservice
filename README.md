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

### Convenciones

- **Base URL**: `http://localhost:<PORT>` (por defecto `8080`).
- **Content-Type (request)**: para `POST`/`PUT`/`PATCH` el servicio exige `Content-Type: application/json` (se valida por prefijo). Si no se envía, responde **400** en formato Problem Details.
- **Content-Type (error)**: los errores usan **RFC 7807** con `Content-Type: application/problem+json`.
- **Envelope de éxito** (endpoints de carrito):
  - `GET /cart/...` → `{ "data": Cart }`
  - `POST /cart/.../items` → `{ "data": CartItem }`
  - Endpoints que devuelven `204 No Content` no retornan body.
- **Probes** (`/livez`, `/readyz`) no usan envelope `data`.
- **Autenticación**: Los endpoints de "Carrito Personal" requieren un token JWT en el header `Authorization: Bearer <token>`. El microservicio recibe el ID de usuario validado a través del Ingress.

### Esquemas (aprox. OpenAPI)

#### Problem (RFC 7807)

```json
{
  "type": "about:blank",
  "title": "Invalid Request Payload",
  "status": 400,
  "detail": "...",
  "instance": "/cart/123/items"
}
```

#### Price

```json
{
  "value": "9.99",
  "updatedAt": "2026-04-15T12:34:56Z"
}
```

- `value`: string decimal con **2** decimales.
- `updatedAt`: timestamp **RFC 3339** en UTC.

#### CartItem

```json
{
  "productId": "fcbcd5d2-7842-40c2-9ee4-c6ff6d99d670",
  "title": "Test Product",
  "unitPrice": { "value": "9.99", "updatedAt": "2026-04-15T12:34:56Z" },
  "quantity": 1,
  "stock": 42,
  "status": "active",
  "addedAt": "2026-04-15T12:34:56Z"
}
```

#### Cart

```json
{
  "userId": "fcbcd5d2-7842-40c2-9ee4-c6ff6d99d670",
  "items": [/* CartItem */],
  "totalPrice": { "value": "29.97", "updatedAt": "2026-04-15T12:34:56Z" }
}
```

### Salud (Probes)

#### `GET /livez`

- **Descripción**: liveness probe (no depende de DB ni servicios externos).
- **Respuestas**:
  - `200 application/json`

    ```json
    {"status":"ok"}
    ```

#### `GET /readyz`

- **Descripción**: readiness probe (hace `Ping` a MongoDB).
- **Respuestas**:
  - `200 application/json`

    ```json
    {"status":"ok"}
    ```

  - `503 application/json` (MongoDB inaccesible)

    ```json
    {"status":"unavailable","detail":"database connection failed"}
    ```

### Carrito Personal (Autenticado)

Estos endpoints operan sobre el carrito del usuario autenticado. El ID de usuario se extrae automáticamente del token de sesión a través del Ingress.

#### `GET /cart`

- **Descripción**: Obtiene el carrito del usuario autenticado.
- **Headers**:
  - `Authorization: Bearer <JWT>`
- **Respuestas**:
  - `200 application/json` → `CartResponse`

    ```json
    {"data": {"userId": "fcbcd5d2-7842-40c2-9ee4-c6ff6d99d670", "items": [], "totalPrice": {"value": "0.00", "updatedAt": "2026-04-15T12:34:56Z"}}}
    ```

  - `401 Unauthorized`
  - `500 application/problem+json`

#### `DELETE /cart`

- **Descripción**: Vacía el carrito del usuario autenticado (idempotente).
- **Headers**:
  - `Authorization: Bearer <JWT>`
- **Respuestas**:
  - `204` (sin body)
  - `401 Unauthorized`
  - `500 application/problem+json`

#### `POST /cart/items`

- **Descripción**: Agrega $n$ unidades del producto al carrito del usuario autenticado.
- **Headers**:
  - `Authorization: Bearer <JWT>`
  - `Content-Type: application/json`
- **Request body**:

  ```json
  {"productId": "fcbcd5d2-7842-40c2-9ee4-c6ff6d99d670", "quantity": 3}
  ```

- **Respuestas**:
  - `201 application/json` → `CartItemResponse`

    ```json
    {"data": {"productId": "fcbcd5d2-7842-40c2-9ee4-c6ff6d99d670", "title": "...", "unitPrice": {"value": "9.99", "updatedAt": "2026-04-15T12:34:56Z"}, "quantity": 3, "stock": 42, "status": "active", "addedAt": "2026-04-15T12:34:56Z"}}
    ```

  - `401 Unauthorized`
  - `404 application/problem+json` (producto inexistente en Catálogo)
  - `422 application/problem+json` (stock insuficiente)
  - `503 application/problem+json` (Catálogo no disponible)
  - `500 application/problem+json`

#### `DELETE /cart/items/{cart_item_id}`

- **Descripción**: Elimina el ítem del carrito personal por ID de producto.
- **Headers**:
  - `Authorization: Bearer <JWT>`
- **Path params**:
  - `cart_item_id` (string, UUID) — *corresponde a `productId`*
- **Respuestas**:
  - `204` (sin body)
  - `401 Unauthorized`
  - `404 application/problem+json` (el producto no está en el carrito)

#### `PUT /cart/increment/{product_id}`

- **Descripción**: Incrementa la cantidad del producto en el carrito personal.
- **Headers**:
  - `Authorization: Bearer <JWT>`
  - `Content-Type: application/json`
- **Path params**:
  - `product_id` (string, UUID)
- **Request body**:

  ```json
  {"quantity": 1}
  ```

- **Respuestas**:
  - `204` (sin body)
  - `401 Unauthorized`
  - `404 application/problem+json` (producto no está en el carrito)
  - `422 application/problem+json` (stock insuficiente)
  - `500 application/problem+json`

#### `PUT /cart/decrement/{product_id}`

- **Descripción**: Decrementa la cantidad del producto en el carrito personal.
- **Headers**:
  - `Authorization: Bearer <JWT>`
  - `Content-Type: application/json`
- **Path params**:
  - `product_id` (string, UUID)
- **Request body**:

  ```json
  {"quantity": 1}
  ```

- **Respuestas**:
  - `204` (sin body)
  - `401 Unauthorized`
  - `404 application/problem+json` (producto no está en el carrito)
  - `500 application/problem+json`

### Carrito (Retrocompatibilidad / Uso Interno)

> [!WARNING]
> Estos endpoints requieren especificar el `user_id` en la URL y están destinados a pruebas o servicios internos. En producción, se recomienda usar los endpoints autenticados de la sección anterior.

#### `GET /cart/{user_id}`

- **Path params**:
  - `user_id` (string, UUID)
- **Respuestas**:
  - `200 application/json` → `CartResponse`

    ```json
    {"data": {"userId": "fcbcd5d2-7842-40c2-9ee4-c6ff6d99d670", "items": [], "totalPrice": {"value": "0.00", "updatedAt": "2026-04-15T12:34:56Z"}}}
    ```

  - `400 application/problem+json` (user_id inválido)
  - `500 application/problem+json`
- **Notas de sincronización**:
  - En cada `GET`, el servicio intenta sincronizar `stock` y `status` contra el microservicio de Catálogo.
  - El `unitPrice` se actualiza **solo si está obsoleto** (más de 24h desde `unitPrice.updatedAt`).
  - Si el Catálogo no está disponible durante el `GET`, se devuelve el carrito con los valores cacheados (best-effort).

#### `DELETE /cart/{user_id}`

- **Descripción**: vacía el carrito (idempotente: vaciar un carrito vacío también es OK).
- **Path params**:
  - `user_id` (string, UUID)
- **Respuestas**:
  - `204` (sin body)
  - `400 application/problem+json`
  - `500 application/problem+json`

#### `POST /cart/{user_id}/items`

- **Descripción**: agrega $n$ unidades del producto al carrito (por defecto $n=1$). Si el producto ya existe, incrementa `quantity` en $n$.
- **Headers**:
  - `Content-Type: application/json`
- **Path params**:
  - `user_id` (string, UUID)
- **Request body**:

  ```json
  {"productId": "fcbcd5d2-7842-40c2-9ee4-c6ff6d99d670", "quantity": 3}
  ```

  - `quantity` (integer, `>= 1`) — opcional, por defecto `1`.

- **Respuestas**:
  - `201 application/json` → `CartItemResponse`

    ```json
    {"data": {"productId": "fcbcd5d2-7842-40c2-9ee4-c6ff6d99d670", "title": "...", "unitPrice": {"value": "9.99", "updatedAt": "2026-04-15T12:34:56Z"}, "quantity": 3, "stock": 42, "status": "active", "addedAt": "2026-04-15T12:34:56Z"}}
    ```

  - `400 application/problem+json` (user_id inválido, JSON inválido o `productId` faltante/incorrecto)
  - `404 application/problem+json` (producto inexistente en Catálogo)
  - `422 application/problem+json` (stock insuficiente)
  - `503 application/problem+json` (Catálogo no disponible)
  - `500 application/problem+json`

#### `DELETE /cart/{user_id}/items/{cart_item_id}`

- **Descripción**: elimina el ítem del carrito por ID de producto del catálogo.
- **Path params**:
  - `user_id` (string, UUID)
  - `cart_item_id` (string, UUID) — *corresponde a `productId`*
- **Respuestas**:
  - `204` (sin body)
  - `400 application/problem+json`
  - `404 application/problem+json` (el producto no está en el carrito)

#### `PUT /cart/{user_id}/{product_id}/increment`

- **Descripción**: incrementa `quantity` del producto en el carrito.
- **Headers**:
  - `Content-Type: application/json`
- **Path params**:
  - `user_id` (string, UUID)
  - `product_id` (string, UUID)
- **Request body**:

  ```json
  {"quantity": 3}
  ```

- **Respuestas**:
  - `204` (sin body)
  - `400 application/problem+json` (path params inválidos o JSON inválido/`quantity` faltante)
  - `404 application/problem+json` (producto no está en el carrito)
  - `422 application/problem+json` (stock insuficiente)
  - `500 application/problem+json`

#### `PUT /cart/{user_id}/{product_id}/decrement`

- **Descripción**: decrementa `quantity` del producto en el carrito. La cantidad **no baja de 1**.
- **Headers**:
  - `Content-Type: application/json`
- **Path params**:
  - `user_id` (string, UUID)
  - `product_id` (string, UUID)
- **Request body**:

  ```json
  {"quantity": 1}
  ```

- **Respuestas**:
  - `204` (sin body)
  - `400 application/problem+json`
  - `404 application/problem+json` (producto no está en el carrito)
  - `500 application/problem+json`

## Tests

El proyecto incluye tests de integración que corren contra una instancia real de MongoDB en Docker.

```bash
# Ejecutar todos los tests
./run_tests.sh
```

## Docker & CI/CD

- **Dockerfile**: Utiliza un build multi-stage para generar una imagen optimizada basada en `alpine`.
- **CI/CD**: Ubicado en `.github/workflows/`, ejecuta tests automáticamente en cada Pull Request y despliega a Kubernetes en cada push a `main`.
