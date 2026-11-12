# API HTTP de Auth Service

Referencia para clientes y SDKs que se comunican con este servicio.

## DTOs

```go
type CredentialsRequest struct {
	Organization string `json:"org_id"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

type TokenResponse struct {
	Token string `json:"token"`
	TTL   int64  `json:"ttl"` // Fecha de expiración Unix en segundos; 0 significa sin expiración.
}

type RegisterResponse struct {
	UUID string `json:"uuid"`
}

type ValidateResponse struct {
	User      string `json:"user"`   // ID de credencial, no el nombre de usuario.
	AppID     string `json:"appId"`
	ExpiresAt int    `json:"exp"`    // Fecha de expiración Unix en segundos; 0 significa sin expiración.
}

type MessageResponse struct {
	Response string `json:"response"`
}

type DeleteUserRequest struct {
	UserID string `json:"userId"` // ID de credencial del usuario que se quiere eliminar.
}

type ErrorResponse struct {
	Error string `json:"error"`
}
```

`CredentialsRequest` se usa en `POST /login` y `POST /register`. `TokenResponse` se devuelve en `POST /login` y `GET /renew`. `MessageResponse` se devuelve en `GET /logout` y `DELETE /user`. `ValidateResponse` es la respuesta de `GET /validate`.

## Operaciones

### Iniciar sesión

`POST /login`

- **Autenticación:** no enviar `Authorization`.
- **Entrada:** `CredentialsRequest`.
- **Éxito (`200`):** `TokenResponse`.
- **Errores:**

| HTTP | `error` |
| --- | --- |
| `400` | `Token provided` si se envía un token en `Authorization`. |
| `400` | `Invalid request body` si el cuerpo no se puede decodificar como JSON. |
| `401` | `Invalid username or password` si las credenciales no son válidas. |
| `410` | `Credential was recently deleted` si la credencial está marcada como eliminada. |
| `500` | `Failed to validate credentials` si falla la comprobación por un error interno. |
| `500` | `Failed to create session` si no se puede crear la sesión. |

### Registrar usuario

`POST /register`

- **Autenticación:** no enviar `Authorization`.
- **Entrada:** `CredentialsRequest`.
- **Éxito (`200`):** `RegisterResponse`.
- **Errores:**

| HTTP | `error` |
| --- | --- |
| `400` | `Token provided` si se envía un token en `Authorization`. |
| `400` | `Invalid request body` si el cuerpo no se puede decodificar como JSON. |
| `500` | `Failed to register user: <detalle>` si falla el registro; `<detalle>` procede del error interno. |

### Validar sesión

`GET /validate`

- **Autenticación:** `Authorization: Bearer <token>`.
- **Entrada:** sin cuerpo.
- **Éxito (`200`):** `ValidateResponse`.
- **Errores:**

| HTTP | `error` |
| --- | --- |
| `400` | `Token not provided` si falta el token. |
| `401` | `Invalid token` si el token no identifica una sesión válida. |
| `410` | `Credential was recently deleted` si la credencial asociada está marcada como eliminada. |
| `500` | `Failed to validate token` si falla una comprobación interna de la credencial. |
| `500` | `Failed to retrieve app ID` si falla la lectura de la sesión. |

### Renovar sesión

`GET /renew`

- **Autenticación:** `Authorization: Bearer <token>`.
- **Entrada:** sin cuerpo.
- **Éxito (`200`):** `TokenResponse` con un token nuevo. El token anterior se invalida.
- **Errores:**

| HTTP | `error` |
| --- | --- |
| `400` | `Token not provided` si falta el token. |
| `409` | `Invalid token` si no se puede recuperar la sesión indicada. |
| `500` | `Error getting credential TTL policy` si no se puede obtener la política de expiración. |
| `500` | `Error creating new session` si no se puede crear la nueva sesión. |

### Cerrar sesión

`GET /logout`

- **Autenticación:** `Authorization: Bearer <token>`.
- **Entrada:** sin cuerpo.
- **Éxito (`200`):** `MessageResponse`; invalida la sesión.
- **Errores:**

| HTTP | `error` |
| --- | --- |
| `400` | `Token not provided` si falta el token. |
| `400` | `API keys cannot be logged out` si el token corresponde a una API key. |
| `409` | `Invalid token` si no se puede recuperar la sesión indicada. |
| `500` | `Error checking if credential is an API key` si falla la comprobación del tipo de credencial. |

### Eliminar usuario

`DELETE /user`

- **Autenticación:** `Authorization: Bearer <token>` de una sesión asociada a la aplicación.
- **Entrada:** `DeleteUserRequest`.

- **Éxito (`200`):** `MessageResponse`. La solicitud de eliminación se publica como evento; el `200` confirma que se aceptó la solicitud, no que el borrado asíncrono haya terminado.
- **Errores:**

| HTTP | `error` |
| --- | --- |
| `400` | `Invalid request body` si el cuerpo no se puede decodificar como JSON. |
| `400` | `Token not provided` si falta el token. |
| `401` | `Invalid token` si el token no corresponde a una sesión conocida. |

## Cliente Go

El cliente recibe su configuración al construirse; no lee variables de entorno por su cuenta. La aplicación puede resolverlas y pasarlas explícitamente:

```go
import (
	"os"

	"github.com/Luis97lol/microservices-sdk/auth"
)

client, err := auth.NewClient(auth.Config{
	BaseURL: os.Getenv("AUTH_BASE_URL"),
})
if err != nil {
	return err
}
```

`Config.HTTPClient` permite inyectar un `*http.Client`, que el cliente configura en el `RequestBuilder` para centralizar el envío; si se omite, se usa `http.DefaultClient`. El cliente expone `Login`, `Register`, `Validate`, `Renew`, `Logout` y `DeleteUser`; todas las operaciones reciben un `context.Context`.

Los errores de las operaciones son `*auth.APIError` (operación, estado HTTP y mensaje del servicio). Las causas de transporte y decodificación se conservan en `Err` y se exponen mediante `Unwrap()`, por lo que `errors.Is` y `errors.As` pueden inspeccionarlas. Los errores de configuración de `NewClient` son `*auth.ConfigError`.
