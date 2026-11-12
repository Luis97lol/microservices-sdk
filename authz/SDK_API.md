# Contrato HTTP para el SDK de Auth

Este documento describe las operaciones HTTP expuestas por el servicio para integraciones externas. Las rutas se publican en la raíz del servidor. El SDK debe recibir la URL base mediante su configuración; el servicio usa `SERVER_PORT` para escuchar y, si no está definido, utiliza `8080`.

## DTOs

Las operaciones de asignación, comprobación y revocación comparten los campos siguientes. Los valores son identificadores opacos para el servicio.

```go
type PermissionScoped struct {
	AppID        string `json:"app_id"`
	CredentialID string `json:"credential_id"`
	PermissionID string `json:"permission_id"`
}
```

`/verify` también decodifica este DTO, pero solo requiere `app_id` y `permission_id`. La identidad de la credencial se obtiene de la sesión indicada en `Authorization`; `credential_id` se ignora en esta operación.

### Respuestas

El protocolo actual no devuelve objetos JSON:

- En caso de éxito, el cuerpo es el texto `OK`.
- En caso de error, el cuerpo es un mensaje de texto plano y el estado HTTP indica el tipo de error.

Como representación interna en el SDK —no como DTO JSON transmitido por la API— se pueden usar:

```go
type OperationResponse struct {
	Body string // "OK" en las respuestas exitosas
}

type APIError struct {
	StatusCode int
	Message    string // cuerpo de texto plano de la respuesta
}
```

## Operaciones

| Operación | Método y endpoint | Campos requeridos | Autenticación |
|---|---|---|---|
| Asignar permiso | `POST /add` | `app_id`, `credential_id`, `permission_id` | No |
| Verificar sesión y permiso | `POST /verify` | `app_id`, `permission_id` | `Authorization` |
| Comprobar permiso | `POST /check` | `app_id`, `credential_id`, `permission_id` | No |
| Revocar permiso | `DELETE /remove` | `app_id`, `credential_id`, `permission_id` | No |

### `POST /add` — asignar permiso

Publica un evento de asignación con los tres identificadores. El handler invoca la publicación antes de responder, pero ignora el posible error devuelto y no informa del resultado de la persistencia.

**Éxito:** `200 OK`, cuerpo `OK`.

**Errores HTTP:**

| Estado | Cuerpo | Causa |
|---|---|---|
| `400 Bad Request` | `Invalid request body` | El cuerpo no se puede decodificar como el DTO. |
| `400 Bad Request` | `Missing required fields` | Falta alguno de los tres campos requeridos o está vacío. |

### `POST /verify` — verificar sesión y permiso

Envía el token de sesión en el encabezado:

```http
Authorization: Bearer <token-de-sesion>
```

El servicio busca la credencial asociada a esa sesión y comprueba que tenga el permiso indicado para la aplicación.

**Éxito:** `200 OK`, cuerpo `OK`.

**Errores HTTP:**

| Estado | Cuerpo | Causa |
|---|---|---|
| `400 Bad Request` | `Invalid request body` | El cuerpo no se puede decodificar como el DTO. |
| `400 Bad Request` | `Missing required fields` | Falta `app_id` o `permission_id`, o está vacío. |
| `401 Unauthorized` | `Token not provided` | No se envió un token en `Authorization`. |
| `401 Unauthorized` | `Unauthorized` | La sesión no se puede resolver o la credencial no tiene ese permiso para la aplicación. |

### `POST /check` — comprobar permiso

Comprueba directamente la asociación entre aplicación, credencial y permiso; no valida un token de sesión.

**Éxito:** `200 OK`, cuerpo `OK`.

**Errores HTTP:**

| Estado | Cuerpo | Causa |
|---|---|---|
| `401 Unauthorized` | `Unauthorized` | El cuerpo es inválido, falta algún campo requerido o el permiso no está concedido. También se devuelve ante un error al consultar el almacenamiento. |

### `DELETE /remove` — revocar permiso

Publica un evento de revocación con los tres identificadores. La publicación se ejecuta en segundo plano, por lo que la respuesta no confirma que la revocación ya se haya persistido.

**Éxito:** `200 OK`, cuerpo `OK`.

**Errores HTTP:**

| Estado | Cuerpo | Causa |
|---|---|---|
| `400 Bad Request` | `Invalid request body` | El cuerpo no se puede decodificar como el DTO. |
| `400 Bad Request` | `Missing required fields` | Falta alguno de los tres campos requeridos o está vacío. |

## Consideraciones para el SDK

- Enviar los cuerpos de las operaciones como JSON con los nombres de campo indicados en las etiquetas `json`.
- No intentar decodificar las respuestas como JSON: leer el cuerpo de texto y usar el estado HTTP para distinguir éxito de error.
- Las comprobaciones de permiso pueden servirse desde una caché interna; los cambios propagados por eventos pueden no reflejarse de forma inmediata.
- `200 OK` en `/add` y `/remove` no garantiza que el cambio haya quedado persistido: `/add` ignora el posible error de publicación y `/remove` publica en segundo plano.

## Cliente Go

El cliente recibe la URL y, opcionalmente, un `*http.Client` mediante `Config`; la aplicación resuelve sus variables de entorno y las pasa al SDK. El cliente expone `AddPermission`, `VerifyPermission`, `CheckPermission` y `RemovePermission`. `VerifyPermission` recibe el token por separado y su DTO solo incluye `app_id` y `permission_id`.

Las operaciones devuelven un `OperationResponse` con el cuerpo de texto. Los errores HTTP son `*APIError`, con estado y cuerpo del servicio; los errores de transporte conservan su causa para `errors.Is` y `errors.As`.
