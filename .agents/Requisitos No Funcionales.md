
# Requisitos No Funcionales

**Red Lines**

Los siguientes incumplimientos **bloquean la evaluación** del proyecto y requieren corrección inmediata:

- **Base de datos compartida entre servicios**: Cada servicio debe tener su propio esquema de persistencia. El acceso directo a la base de datos de otro servicio no está permitido.
- **Pipeline de CI roto**: La rama principal debe estar en estado ejecutable en todo momento. Un CI en rojo que no se atiende en la sesión siguiente es un bloqueante.
- **Secretos en el código fuente o historial de Git**: Cualquier credencial, token o secret que aparezca en el repositorio (incluso en commits anteriores) requiere rotación inmediata y limpieza del historial.

## Componentes

La plataforma debe construirse como un conjunto de servicios con responsabilidades bien delimitadas y bases de datos independientes. La identificación de los dominios, la descomposición en servicios y los contratos entre ellos quedan a criterio del grupo y deben justificarse mediante decisiones de arquitectura documentadas. Se espera que el diseño garantice consistencia ante fallos parciales en los flujos que involucran múltiples servicios.

Cada grupo debe desarrollar:

- Un conjunto de **servicios backend**.
- Una **aplicación mobile** (React Native).
- Un **backoffice web** para administración.
- Un **API Gateway** como punto único de entrada al sistema, que centralice el enrutamiento hacia los servicios backend, la validación de tokens de sesión y el rate limiting. Su diseño e implementación deben justificarse en un ADR. En caso excepcional de no adoptarlo, el grupo debe documentar el mecanismo alternativo con la misma rigurosidad y justificar la decisión.

**Tecnologías**

- Servicios backend: **lenguaje a elección del grupo**; se sugiere Python, Node.js o Go por ser los lenguajes donde la cátedra tiene más expertise para acompañar. Frameworks y bases de datos libres (se recomienda PostgreSQL y/o MongoDB).
- Aplicación mobile: React Native (recomendado).
- Backoffice web: React (recomendado).

La elección tecnológica debe quedar justificada en la documentación técnica del proyecto.

## UX

La experiencia de usuario es parte del producto. Antes de escribir código de interfaz, el equipo debe pensar cómo se ve, cómo se navega y por qué. Se espera:

- **Prototipado**: wireframes o mockups de los flujos principales (se recomienda Figma o equivalente), incluyendo la app mobile y el backoffice. Pueden apoyarse en herramientas como Stitch de Google o similares, y en sistemas de diseño ya probados.
- **Sistema de diseño**: elegir un sistema de diseño (ej. Material Design, Apple HIG u otro) y mantenerlo consistente en toda la interfaz.
- **Decisiones de UX**: cómo se comporta el usuario en la aplicación, cómo se ve y cómo se navega son decisiones creativas del equipo. Lo importante es que puedan explicar por qué tomaron esas decisiones.
- **Feedback temprano**: los prototipos deben presentarse en el Checkpoint 1. A partir de ahí, el diseño se itera en las weeklies con acompañamiento del ayudante.
- **Accesibilidad básica**: las interfaces deben contemplar contraste de color legible y tamaños de texto adecuados para uso en dispositivos móviles.

## Integridad y Flujo de Datos

Los flujos que involucran múltiples servicios deben garantizar consistencia ante fallos parciales. En particular:

- **Consistencia distribuida**: Si un flujo toca varios servicios y uno falla, los pasos anteriores no pueden quedar en un estado inconsistente. El grupo decide la estrategia y la documenta en un ADR. Por ejemplo, si el pago se confirma pero falla la creación de la orden, el sistema debe revertir o compensar el cobro.
- **Idempotencia**: Las operaciones donde duplicar tiene consecuencias graves (cobrar dos veces, descontar stock, crear dos órdenes) deben ser idempotentes: si el mismo request llega dos veces, el resultado es el mismo sin repetir side effects. No todo necesita idempotencia explícita — un update de perfil ya es idempotente, un registro duplicado se resuelve con unique constraint. La clave explícita es para flujos donde el reintento es esperable y duplicar duele.
- **Manejo de errores distribuidos**: Los servicios deben diferenciar entre fallos transitorios (reintentar) y permanentes (compensar o notificar). No se acepta silenciar errores ni propagar estados inconsistentes. La definición concreta de qué constituye un fallo transitorio o permanente en cada flujo queda a criterio del grupo y debe acordarse con el corrector asignado.
- **Eventos y asincronía**: Si se adopta comunicación asíncrona (colas de mensajes), los consumidores deben ser idempotentes y el sistema debe manejar mensajes duplicados o fuera de orden.
- **Resiliencia en la comunicación entre servicios**: Las llamadas entre servicios deben contemplar estrategias ante fallos transitorios. Se recomienda aplicar patrones como Retry (con backoff exponencial) y Circuit Breaker para evitar cascadas de fallo. La elección y justificación de estos patrones debe quedar documentada en los ADRs correspondientes.
- **Resiliencia en la capa de presentación**: La aplicación mobile y el backoffice deben manejar explícitamente los estados de error y latencia: indicadores de carga, mensajes de error accionables ante fallos de red, y comportamiento coherente ante respuestas parciales o timeout del backend. No se acepta que un fallo de red deje la interfaz en un estado inconsistente o sin feedback al usuario.

## Testing

El desarrollo debe incluir:

- **Pruebas unitarias** por servicio.
- **Pruebas de integración** entre servicios core.
- **Contratos de API** entre servicios definidos con OpenAPI/Swagger como fuente de verdad para la comunicación. Se valora positivamente la verificación automatizada de estos contratos (ej: Pact, Schemathesis, Dredd).
- **Pruebas de carga y estrés**: ejecutar al menos un escenario de carga simulando usuarios concurrentes sobre los endpoints críticos. Documentar los resultados, cuellos de botella identificados y acciones tomadas (se recomienda k6 o Artillery).
- Cobertura de tests superior al **70%** en los servicios backend.

## Gestión de Ingeniería

- **Trazabilidad backlog-repositorio**: Cada pull request debe referenciar la historia o tarea que lo origina. Se esperan mensajes de commit descriptivos; no se exige un formato estricto ni referencia a tickets en cada commit individual, pero no se aceptan commits genéricos sin contexto ("fix", "changes", "wip").
- **Atomicidad en Git**: Cada pull request debe representar un cambio lógico coherente. No deben mezclarse features no relacionadas en un mismo PR.
- **Uso de herramientas de IA**: Se espera que los equipos utilicen herramientas de asistencia con IA como parte natural del desarrollo. Lo que se evalúa no es si se usó IA, sino la calidad de la dirección: evidencia de flujos estructurados de trabajo con IA (ej.`AGENTS.md`, `.agents/skills` o equivalentes), y **capacidad de cada integrante de explicar y defender cualquier parte del código en la defensa del proyecto**. Las decisiones de arquitectura deben ser del equipo, no defaults aceptados sin criterio; los ADRs deben reflejar alternativas evaluadas. El equipo es responsable final del código independientemente de la herramienta que lo produjo.
- **Backlog actualizado**: El backlog debe reflejar el estado real del proyecto en cada sesión de seguimiento. Deben existir tareas explícitas para trabajo de infraestructura, testing e integración, no solo para funcionalidades de negocio.

## Despliegue en la Nube

Los servicios deben desplegarse en la nube utilizando una **plataforma como servicio (PaaS)**, o un **orquestador de contenedores gestionado** o directamente un **cloud provider** para grupos que prefieran mayor control de infraestructura. La elección queda a criterio del grupo y debe estar documentada y justificada en un ADR.

Se debe elegir una alternativa que cuente con un plan gratuito o de bajo costo vigente. Para un análisis detallado de las opciones disponibles (proveedores, free tiers, stacks recomendados y consejos de presupuesto), consultar la **[Guía de cátedra: Cloud Computing](https://ingenieria-del-software-2.github.io/blog/2025-09-11-cloud-computing/)**.

Se debe garantizar que el entorno de producción sea reproducible y que el despliegue ocurra de forma automatizada desde el pipeline de CI/CD.

**Despliegue usando k8s**

Como alternativa avanzada, el grupo puede optar por desplegar la plataforma en Kubernetes. Esta decisión debe estar justificada y documentada.

**Empaquetamiento**

- Servicios backend: `Dockerfile` + archivo de dependencias correspondiente al lenguaje elegido.
- Aplicación mobile: APK o equivalente.
- Entorno local: se debe poder levantar el entorno completo de forma reproducible (se recomienda `docker-compose`).

**CI-CD**

- CI: correr automáticamente ante cada push y **bloquear el merge** ante fallas en los tests. Se recomienda incorporar análisis estático de seguridad (SAST).
- CD: despliegue automático al entorno productivo desde el branch principal o mediante tags, documentado.

**Gestión de imágenes y media**

Los archivos multimedia deben almacenarse en un servicio externo de objetos (ej. AWS S3, Google Cloud Storage, Cloudinary o equivalente). No guardar binarios en la base de datos ni en el filesystem del servidor; solo se persiste la URL.

- Imágenes de perfil: JPEG, PNG o WebP, hasta 5 MB.
- Imágenes de producto: JPEG, PNG o WebP, hasta 10 MB por archivo.
- Videos de producto _(opcional)_: MP4 o WebM, hasta 50 MB.

Otros formatos deben rechazarse con un error descriptivo. Las URLs deben ser públicamente accesibles. Se recomienda limpiar archivos huérfanos al reemplazar o eliminar media.

## Observabilidad

**Monitoreo**

Cada servicio debe exponer dos endpoints con semántica diferenciada:

- **`/livez`**: valida que el proceso está vivo. Solo verificaciones locales (ej: el servidor HTTP responde). No debe consultar base de datos ni servicios externos.
- **`/readyz`**: valida que el servicio puede operar correctamente. Debe verificar dependencias críticas (ej: base de datos). Un servicio que responde HTTP pero no puede cumplir su funcionalidad principal no es ready.

**`/livez` y `/readyz` no son equivalentes**: no deben compartir lógica. Un endpoint que responde `200 OK` sin validar lo que corresponde a su semántica no cumple. No usar un único `/health` genérico.

**Cada servicio debe contar además con un sistema de observabilidad** (APM o equivalente) para supervisar su estado operativo y facilitar el diagnóstico de fallas. Se valora incorporar **trazabilidad distribuida** con un trace/correlation ID propagado entre servicios.

**Logs**

Cada servicio debe contar con log estructurado con niveles configurables:

- **Error**: falla que impide ejecutar la funcionalidad.
- **Warn**: condición anómala que no impide la funcionalidad básica.
- **Info**: acción iniciada por el usuario o el sistema, trazable en el historial.
- **Debug**: contexto técnico para diagnóstico sin necesidad del código fuente.

## Seguridad

- Las contraseñas deben almacenarse hasheadas (BCrypt o equivalente).
- Los secretos y credenciales no deben estar en el código fuente; deben gestionarse mediante variables de entorno o un sistema de secretos.
- **Protección de datos personales**: Los datos sensibles de los usuarios (email, dirección, historial de compras) solo deben recolectarse si son estrictamente necesarios para la funcionalidad. No deben exponerse en logs, respuestas de error ni endpoints no autorizados. El acceso a datos de otros usuarios debe estar controlado por autorización explícita en cada operación.
- **Minimización de datos en APIs**: Las respuestas de la API no deben incluir campos innecesarios. En particular, nunca deben retornarse contraseñas, tokens internos ni datos privados de otros usuarios, aunque estén disponibles en la base de datos.

**Lineamientos técnicos de autenticación**

- **Tokens de sesión**: Se recomienda el uso de JWT con tiempo de expiración configurable. El token debe incluir el identificador de usuario y el rol.
- **Tokens de recupero**: Deben generarse con al menos 128 bits de entropía (ej. UUID v4 o equivalente), almacenarse hasheados en la base de datos y expirar en máximo 1 hora.
- **Contraseñas**: Deben cumplir mínimo 8 caracteres e incluir al menos una mayúscula, una minúscula y un número. El almacenamiento debe usar BCrypt u otro algoritmo de hashing adaptativo.
- **Refresh de sesión**: Se debe soportar renovación de token sin requerir re-autenticación mientras la sesión sea válida.
- **Rate limiting**: Los endpoints de login y recupero deben tener límite de intentos por IP y por cuenta.

## Documentación

Se debe entregar:

- **README** por repositorio con instrucciones de setup local.
- **Contratos de API** documentados con Swagger/OpenAPI, preferentemente autogenerado.
- **Diagramas de arquitectura** (se recomienda C4 Model: contexto, contenedores y componentes).
- **Registro de Decisiones de Arquitectura (ADRs)** para las decisiones de diseño relevantes. Cada ADR debe incluir: contexto del problema, decisión tomada, alternativas descartadas y consecuencias esperadas.
- **Retrospectiva final del proyecto**.

Se valora positivamente:

- **Guía de usuario** de la aplicación mobile.
- **Guía de usuario** del backoffice.
- **Bitácora de proyecto** con evolución semanal del backlog.

---

# Instancias de entrega

El proyecto se desarrolla en forma iterativa e incremental a lo largo de tres checkpoints y una entrega final. Cada instancia dura aproximadamente 3–4 semanas.


**Dinámica de trabajo**

El corrector asignado a cada grupo actúa como **Product Owner** durante todo el cuatrimestre. Al inicio de cada checkpoint, el grupo acuerda con el corrector el conjunto de historias de usuario a trabajar en ese período. Ese acuerdo queda registrado en el backlog del grupo.

Durante el checkpoint, el grupo trabaja semanalmente de forma iterativa. En cada sesión semanal el corrector evalúa el avance funcional y el estado de las distintas [dimensiones de calidad](#requisitos-no-funcionales) del proyecto. Al finalizar el período, el grupo realiza la entrega formal de las historias acordadas.

**Instancias**

| Instancia    | Duración aprox. | Fecha      | Recomendaciones                                                                                                                                          |
| ------------ | --------------- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Checkpoint 0 | —               | 27/03/2026 | Organizar el equipo, reconocer componentes, conocer herramientas. Establecer una matriz de conocimientos del equipo                                      |
| Checkpoint 1 | 3–4 semanas     | 17/04/2026 | Diseñar arquitectura, armar repositorios, automatizar CI/CD en todos los componentes. Al menos un CRUD básico funcional. Presentar prototipos UX (wireframes/mockups) de los flujos principales |
| Checkpoint 2 | 3–4 semanas     | 12/05/2026 | E2E de la plataforma con todos los CRUD básicos al 100% o avanzados. 70% historias requeridas finalizadas, 50% historias optativas finalizadas            |
| Checkpoint 3 | 3–4 semanas     | 05/06/2026 | 100% historias requeridas finalizadas, 80% historias optativas finalizadas                                                                               |
| Entrega      | —               | 26/06/2026 | Trabajo completo más demo funcional                                                                                                                      |

_Las recomendaciones son orientativas y deben revisarse con el ayudante docente asignado al inicio de cada período. El alcance específico de cada checkpoint se acuerda con el corrector._

**Al finalizar el trabajo**, el grupo debe haber implementado:

- **Todas las historias obligatorias**, cumpliendo sus criterios de aceptación.
- **Todos los Requisitos No Funcionales** definidos en este documento.
- **Un mínimo de puntos en historias optativas** según la cantidad de integrantes del grupo (ver [Historias optativas](#historias-optativas)).