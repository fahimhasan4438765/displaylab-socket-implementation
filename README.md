# displaylab-socket

Separate Gin service that runs alongside DisplayLab.

When DisplayLab's FCM send path fires (screen/playlist updates, etc.), PHP also POSTs the **same payload** here. Connected Flutter/Android players receive it as a WebSocket `screen_update` event and can refresh their display list.

No database. Token matching only.

## Endpoints

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| `GET` | `/health` | — | Health + connection counts |
| `GET` | `/ws?device_token=<fcm_token>` | — | Device WebSocket subscribe |
| `POST` | `/internal/v1/notify` | `X-Internal-Key` | Called by DisplayLab PHP |

### Notify body (same shape as FCM message fields)

```json
{
  "token": "<device FCM token / add_screen.device_code>",
  "title": "Display Lab",
  "body": "Playlist Updated",
  "data": {
    "playlist_update": "true"
  }
}
```

### Socket event delivered to device

```json
{
  "event": "screen_update",
  "token": "<same token>",
  "title": "Display Lab",
  "body": "Playlist Updated",
  "data": {
    "playlist_update": "true"
  }
}
```

## Run

```bash
cp .env.example .env
go mod tidy
make run
# listens on :8090
```

## DisplayLab wiring

PHP `pushNotification.php` (all copies) calls:

`POST http://127.0.0.1:8090/internal/v1/notify`

Configured in `application/config/config.php`:

- `$config['socket_notify_url']`
- `$config['socket_notify_key']`

FCM still sends as before; socket notify is additive and non-blocking (failures are ignored so CMS actions are not broken).
