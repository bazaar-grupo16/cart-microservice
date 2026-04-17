# Contexto Técnico Adicional - Infraestructura, Ruteo y Autenticación (Bazaar)

Este documento detalla las decisiones arquitectónicas tomadas para la gestión del código fuente, el enrutamiento de tráfico (API Gateway) y la validación de sesiones, apuntando a cumplir con los Requisitos No Funcionales (RNF) del proyecto Bazaar.

---

## 1. Estrategia de Gestión de Repositorios

Para garantizar la independencia de los microservicios y mantener historiales de Git limpios, se definió la siguiente política de creación de repositorios:

* **Decisión:** NO se utilizarán "Forks" del repositorio base. Se utilizará la funcionalidad **"GitHub Template Repositories"**.
* **Justificación:** * Los Forks dentro de una misma Organización de GitHub presentan limitaciones técnicas.
  * Un Fork mantiene un vínculo (upstream) con la plantilla, lo que puede provocar Pull Requests accidentales hacia el repositorio base.
  * Usar la plantilla genera un repositorio nuevo con un historial de Git en blanco, ideal para iniciar un microservicio independiente.
* **Procedimiento Operativo:** Al crear un microservicio desde el Template, el desarrollador DEBE actualizar manualmente el nombre del servicio, las etiquetas (`labels`), y la URL de la imagen de GHCR en los manifiestos de Kubernetes (`deployment.yaml`, `service.yaml`) y en el pipeline de CI/CD.

---

## 2. Arquitectura del API Gateway

Para cumplir con el RNF que exige un punto único de entrada y centralización del ruteo y autenticación, se optó por una solución nativa de Kubernetes.

* **Decisión:** Se utilizará **NGINX Ingress Controller** instalado a nivel clúster, gestionado vía Helm.
* **Infraestructura Cloud:** Se aprovisionará un único **Load Balancer** en DigitalOcean que expondrá el Ingress Controller y proveerá la IP pública del sistema.
* **Separación de Namespaces (Blast Radius):** * El Ingress Controller se instalará en un namespace dedicado (ej. `ingress-nginx`).
  * Los microservicios vivirán en su propio namespace (`bazaar-dev`).
  * **Justificación:** Esto protege el Load Balancer y la IP pública en caso de que el entorno de desarrollo sea destruido o recreado, y permite aplicar políticas de seguridad (RBAC) separadas.

---

## 3. Repositorio de Infraestructura (`bazaar-infra`)

Se adopta el patrón de Infraestructura como Código (IaC) centralizada para los componentes compartidos.

* **Decisión:** Se creó un repositorio dedicado llamado `bazaar-infra`.
* **Propósito:** Alojar los pipelines de GitHub Actions y los archivos de configuración (`values.yaml`) necesarios para instalar y actualizar componentes globales del clúster (como el NGINX Ingress Controller mediante Helm), sin mezclar esta lógica con el código de negocio de los microservicios.

---

## 4. Autenticación y Validación de Sesiones Centralizada

Para cumplir con la validación de tokens sin acoplar la lógica de seguridad en cada microservicio de negocio (como Órdenes o Productos), se utilizará el patrón *API Gateway Token Validation*.



* **Flujo de Autenticación:**
  1. Se desarrollará un microservicio específico de identidades (`auth-service`).
  2. Los microservicios de negocio definirán sus rutas de entrada mediante archivos `ingress.yaml` en sus propios repositorios.
  3. Para proteger una ruta, se utilizará la anotación nativa de NGINX: `nginx.ingress.kubernetes.io/auth-url`.
  4. NGINX interceptará las peticiones entrantes y realizará una sub-petición interna al `auth-service` para validar el JWT.
  5. Solo si el `auth-service` responde con HTTP 200, NGINX enrutará la petición original al microservicio de destino; de lo contrario, retornará HTTP 401.