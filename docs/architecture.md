# 🏛️ Mediguk Context Engine - Architecture

## 🔐 Modelo de Seguridad: Split-Token

Para prevenir ataques de **XSS** y **CSRF** simultáneamente, utilizamos este patrón:

| Componente      | Ubicación                | Protege contra...                                          |
| :-------------- | :----------------------- | :--------------------------------------------------------- |
| **JWT**         | Cabecera `Authorization` | **CSRF**: No se envía automáticamente, requiere JS manual. |
| **Fingerprint** | Cookie `HttpOnly`        | **XSS**: El Javascript malicioso no puede leer la cookie.  |

> [!TIP]
> Go valida ambos de forma **Stateless** (sin base de datos) usando criptografía asimétrica **Ed25519**, lo que garantiza un rendimiento extremo.
