# Catalog Microservice

## Endpoints

### Productos

#### `POST /products` — Crear producto

Crea el producto en catalog y registra su stock inicial en el inventory service.

**Body (multipart/form-data):**

| Campo | Tipo | Restricciones |
|-------|------|---------------|
| `title` | string | 1–150 caracteres, no vacío |
| `description` | string | mínimo 1 carácter, no vacío |
| `price` | float | ≥ 0.01 |
| `stock` | int | ≥ 0 — se envía al inventory service |
| `category` | string | 1–80 caracteres (se normaliza a minúsculas) |
| `seller_id` | UUID | — |
| `files` | imágenes | Al menos una imagen (JPEG, PNG o WebP, máx. 10 MB c/u) |

**Respuesta `201`:**
```json
{
  "data": {
    "id": "uuid",
    "title": "Campera de cuero",
    "description": "Campera negra talle M",
    "price": 150.0,
    "stock": 10,
    "category": "indumentaria",
    "sellerId": "uuid",
    "status": "active",
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

---

#### `GET /products` — Listar productos

Devuelve todos los productos activos. El campo `stock` de cada producto se obtiene del inventory service. Los productos con stock 0 se incluyen en los resultados — el frontend decide cómo mostrarlos visualmente.

**Query params:**

| Parámetro | Tipo | Default | Descripción |
|-----------|------|---------|-------------|
| `limit` | int (1–50) | `20` | Cantidad máxima de resultados |
| `offset` | int (≥0) | `0` | Desplazamiento para paginación |
| `search_query` | string | — | Búsqueda por texto libre en título y descripción |
| `category` | string | — | Filtrar por categoría |
| `seller_id` | UUID | — | Filtrar por vendedor |
| `include_inactive` | bool | `false` | Incluir productos con status `inactive` |
| `sort_by` | `price` \| `created_at` \| `title` | — | Campo de ordenamiento |
| `order` | `asc` \| `desc` | `asc` | Dirección del ordenamiento |

**Respuesta `200`:**
```json
{
  "data": [ /* array de productos */ ],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

---

#### `GET /products/{id}` — Obtener producto por ID

El `stock` devuelto se consulta en tiempo real al inventory service.

**Respuesta `200`:** objeto `ProductResponse` (mismo formato que al crear).  
**Respuesta `404`:** error en formato `application/problem+json`.

---

#### `PATCH /products/{id}` — Actualizar producto parcialmente

Solo se modifican los campos presentes en el body. Si se incluye `stock`, el cambio se delega al inventory service.

| Campo | Tipo | Restricciones |
|-------|------|---------------|
| `title` | string | 1–150 caracteres, no vacío |
| `description` | string | 1–150 caracteres, no vacío |
| `price` | float | ≥ 0.01 |
| `stock` | int | ≥ 0 — se actualiza en el inventory service |
| `category` | string | 1–150 caracteres (se normaliza a minúsculas) |
| `status` | `active` \| `inactive` | — |

**Respuesta `200`:** objeto `ProductResponse` con los datos actualizados.  
**Respuesta `404`:** error en formato `application/problem+json` si el producto no existe.

---

#### `DELETE /products/{id}` — Eliminar producto

Elimina el producto de catalog, borra sus imágenes de DigitalOcean Spaces y elimina su entrada de stock en el inventory service.

**Respuesta `204`:** sin contenido.  
**Respuesta `404`:** error en formato `application/problem+json` si el producto no existe.

---

#### `POST /products/{id}/images` — Agregar imágenes

**Body (multipart/form-data):** uno o más archivos bajo el campo `files` (JPEG, PNG o WebP, máx. 10 MB c/u).

**Respuesta `200`:** objeto `ProductResponse` con el array `images` actualizado.

---

#### `DELETE /products/{id}/images` — Eliminar imágenes

**Body JSON:**
```json
{ "urls": ["https://cdn.../imagen.jpg"] }
```
No se pueden eliminar todas las imágenes de un producto activo.

**Respuesta `200`:** objeto `ProductResponse` con el array `images` actualizado.  
**Respuesta `409`:** si la operación dejaría el producto activo sin imágenes.

---

#### `PATCH /products/{id}/images/order` — Reordenar imágenes

**Body JSON:**
```json
{ "urls": ["https://cdn.../b.jpg", "https://cdn.../a.jpg"] }
```
Debe contener exactamente las mismas URLs que el producto, en el nuevo orden deseado.

**Respuesta `200`:** objeto `ProductResponse` con el nuevo orden de imágenes.

---

### Formato de errores

Todos los errores de dominio siguen [RFC 7807 Problem Details](https://www.rfc-editor.org/rfc/rfc7807) con `Content-Type: application/problem+json`:

```json
{
  "type": "about:blank",
  "title": "Not Found",
  "status": 404,
  "detail": "Product not found",
  "instance": "/products/550e8400-e29b-41d4-a716-446655440000"
}
```
