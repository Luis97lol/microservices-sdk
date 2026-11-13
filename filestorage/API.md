# API HTTP de FileStorage

Referencia de las operaciones HTTP disponibles para clientes externos.

## DTOs

Las operaciones no reciben un DTO JSON. La carga se envía como `multipart/form-data`; el módulo y el identificador, cuando se usan, forman parte de la ruta.

```go
import "mime/multipart"

// UploadRequest representa el único campo multipart aceptado por /upload.
type UploadRequest struct {
	File *multipart.FileHeader `form:"file"`
}

type UploadResponse struct {
	Module string `json:"module"`
	FileID string `json:"fileId"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
```

En las respuestas:

- La carga devuelve `UploadResponse` en JSON. `fileId` es el ID suministrado en la ruta o un UUIDv4 generado por el servicio.
- La descarga devuelve el contenido binario del archivo, no un DTO JSON.
- El borrado correcto no devuelve cuerpo (`204 No Content`).
- Los errores con cuerpo devuelven `ErrorResponse`. El texto de `error` es sensible al mensaje interno y no constituye un código estable.

## Autenticación y ámbito

Todas las operaciones requieren el encabezado `Authorization` con un token de sesión que el servicio de autenticación pueda validar (por ejemplo, `Bearer <token>`). El servicio reenvía el encabezado al endpoint de validación de Auth Service. No hay un endpoint de inicio de sesión en FileStorage.

El `appId` no se envía en la ruta ni en el cuerpo: se obtiene de la sesión validada. El servicio comprueba además el permiso de carga, descarga o borrado correspondiente antes de realizar cada operación.

`module` es un segmento opcional para agrupar archivos. `fileId` es un UUIDv4. En la carga se puede omitir; en ese caso el servicio genera uno. Si se proporciona un `module`, el servicio elimina barras, secuencias `..` y espacios al inicio/final al construir la ruta de almacenamiento.

## Operaciones

### Subir un archivo

- `POST /upload`
- `POST /upload/{fileId}`
- `POST /{module}/upload`
- `POST /{module}/upload/{fileId}`

- **Autenticación:** encabezado `Authorization` requerido.
- **Entrada:** `multipart/form-data` con una parte de archivo llamada `file`. No se admite un cuerpo JSON.
- **Parámetros de ruta:** `module` y `fileId` son opcionales según la ruta elegida.
- **Éxito (`200`):** `UploadResponse`.
- **Errores:**

| HTTP | `error` | Cuándo ocurre |
| --- | --- | --- |
| `400` | `Max file size exceeded (50 MB)` | No se pudo procesar el formulario multipart. |
| `400` | `No file uploaded` | Falta la parte `file` o no se pudo abrir. |
| `400` | `Empty file` | El archivo recibido tiene tamaño cero. |
| `400` | `invalid appId: <detalle>`, `invalid fileId: <detalle>` o `app "<appId>" not found in config` | El `appId` resuelto o el ID proporcionado no es UUIDv4, o la aplicación no está configurada. |
| `401` | `Unauthorized` | El token no se pudo validar. |
| `503` | `Unauthorized` | El permiso de carga fue denegado o el servicio de permisos no respondió correctamente. |
| `503` | Mensaje del servicio | El backend de almacenamiento no está disponible. |
| `500` | Mensaje del servicio | Otro error interno al resolver o guardar el archivo. |

El handler configura `50 MiB` como umbral de memoria de `ParseMultipartForm` y responde con el mensaje indicado si el parseo falla; actualmente no establece un límite duro de tamaño del cuerpo.

### Descargar un archivo

- `GET /download/{fileId}`
- `GET /{module}/download/{fileId}`

- **Autenticación:** encabezado `Authorization` requerido.
- **Entrada:** sin cuerpo. `fileId` es obligatorio.
- **Éxito (`200`):** bytes del archivo con `Content-Type: application/octet-stream`. Si la solicitud anuncia `Accept-Encoding: gzip`, la respuesta se comprime y lleva `Content-Encoding: gzip`.
- **Errores:**

| HTTP | `error` | Cuándo ocurre |
| --- | --- | --- |
| `400` | `invalid appId: <detalle>`, `invalid fileId: <detalle>` o `app "<appId>" not found in config` | El `appId` resuelto o el ID proporcionado no es UUIDv4, o la aplicación no está configurada. |
| `401` | `Unauthorized` | El token no se pudo validar. |
| `503` | `Unauthorized` | El permiso de descarga fue denegado o el servicio de permisos no respondió correctamente. |
| `503` | Mensaje del servicio | El backend de almacenamiento no está disponible. |
| `500` | Mensaje del servicio | Otro error al resolver o descargar el archivo. |

### Eliminar un archivo

- `DELETE /delete/{fileId}`
- `DELETE /{module}/delete/{fileId}`

- **Autenticación:** encabezado `Authorization` requerido.
- **Entrada:** sin cuerpo. `fileId` es obligatorio.
- **Éxito (`204`):** sin cuerpo.
- **Errores:**

| HTTP | `error` | Cuándo ocurre |
| --- | --- | --- |
| `400` | `invalid appId: <detalle>`, `invalid fileId: <detalle>` o `app "<appId>" not found in config` | El `appId` resuelto o el ID proporcionado no es UUIDv4, o la aplicación no está configurada. |
| `401` | `Unauthorized` | El token no se pudo validar. |
| `503` | `Unauthorized` | El permiso de borrado fue denegado o el servicio de permisos no respondió correctamente. |
| `503` | Mensaje del servicio | El backend de almacenamiento no está disponible. |
| `500` | Mensaje del servicio | Otro error al resolver o borrar el archivo. |

## Consideraciones para el cliente

- Los IDs de archivo deben tratarse como UUIDv4 y conservarse de la respuesta de subida para posteriores descargas o borrados.
- El servicio distingue el ámbito por la aplicación derivada del token, el módulo y el ID del archivo. Un mismo `fileId` en otra aplicación o módulo no apunta al mismo objeto.
- Los errores de autenticación/permisos usan el mismo formato `{"error":"..."}`. En los errores de almacenamiento, `error` puede contener detalles variables del backend.
- Las rutas aceptan `OPTIONS` para solicitudes CORS.

## Cliente Go

La aplicación proporciona la URL base al construir el cliente; el cliente no lee variables de entorno por su cuenta. `Config.HTTPClient` es opcional; si no se proporciona, se usa `http.DefaultClient`.

```go
import (
	"context"
	"os"

	"github.com/Luis97lol/microservices-sdk/filestorage"
)

func uploadAndDownload() error {
	file, err := os.Open("report.pdf")
	if err != nil {
		return err
	}
	defer file.Close()

	client, err := filestorage.NewClient(filestorage.Config{
		BaseURL: os.Getenv("FILESTORAGE_BASE_URL"),
	})
	if err != nil {
		return err
	}

	ctx := context.Background()
	sessionToken := os.Getenv("SESSION_TOKEN")
	uploaded, err := client.Upload(ctx, sessionToken, filestorage.UploadRequest{
		Module:   "reports",
		Filename: "report.pdf",
		Content:  file,
	})
	if err != nil {
		return err
	}

	downloaded, err := client.Download(ctx, sessionToken, uploaded.Module, uploaded.FileID)
	if err != nil {
		return err
	}
	defer downloaded.Body.Close()
	return nil
}
```

`UploadRequest.Content` se transmite como streaming multipart, sin acumular el archivo completo en memoria. `DownloadResponse.Body` también es un stream y el llamador debe cerrarlo. `DeleteFile` devuelve `nil` para `204 No Content`. Los errores de servicio y de transporte son `*filestorage.APIError`; `errors.As` permite inspeccionar operación, estado y mensaje, y `Unwrap()` conserva las causas originales.
