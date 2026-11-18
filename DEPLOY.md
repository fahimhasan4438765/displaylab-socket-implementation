# Deploy displaylab-socket on a VPS

Recommended layout when DisplayLab PHP already runs on the same VPS:

- Go service listens on `127.0.0.1:8090`
- Nginx (or Apache) proxies public `wss://your-domain/ws` to it
- PHP calls `http://127.0.0.1:8090/internal/v1/notify` (private, not exposed)

---

## 1) Server prerequisites

Ubuntu/Debian example:

```bash
sudo apt update
sudo apt install -y git nginx
# optional if you build on the VPS:
# sudo apt install -y golang-go
```

Create a user/dir:

```bash
sudo useradd -r -s /usr/sbin/nologin displaylab-socket || true
sudo mkdir -p /opt/displaylab-socket
sudo chown $USER:$USER /opt/displaylab-socket
```

---

## 2) Build the binary

### Option A — build on your Mac, upload binary (simple)

On your Mac:

```bash
cd /Users/user/Documents/GitHub/displaylab-socket
GOOS=linux GOARCH=amd64 go build -o bin/displaylab-socket-linux ./cmd/server
# if VPS is ARM: GOARCH=arm64
scp bin/displaylab-socket-linux user@YOUR_VPS:/opt/displaylab-socket/displaylab-socket
scp .env.example user@YOUR_VPS:/opt/displaylab-socket/.env
```

### Option B — build on the VPS

```bash
cd /opt/displaylab-socket
git clone <your-displaylab-socket-repo-url> .
# install Go 1.22+ if needed, then:
go build -o displaylab-socket ./cmd/server
cp .env.example .env
```

---

## 3) Configure `.env` on VPS

```bash
nano /opt/displaylab-socket/.env
```

```env
HTTP_ADDR=127.0.0.1:8090
INTERNAL_API_KEY=CHANGE_ME_TO_A_LONG_RANDOM_SECRET
WS_ALLOW_ORIGINS=https://displaylab.net,https://your-app-domain.com
```

Notes:

- Bind to `127.0.0.1` so notify is not public.
- Use a strong `INTERNAL_API_KEY` (same value in PHP config).
- Restrict `WS_ALLOW_ORIGINS` in production if Flutter/web clients send Origin.

---

## 4) systemd service (auto-restart)

```bash
sudo tee /etc/systemd/system/displaylab-socket.service >/dev/null <<'EOF'
[Unit]
Description=DisplayLab Socket Service
After=network.target

[Service]
Type=simple
User=displaylab-socket
Group=displaylab-socket
WorkingDirectory=/opt/displaylab-socket
EnvironmentFile=/opt/displaylab-socket/.env
ExecStart=/opt/displaylab-socket/displaylab-socket
Restart=always
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

sudo chown -R displaylab-socket:displaylab-socket /opt/displaylab-socket
sudo chmod 750 /opt/displaylab-socket
sudo chmod 600 /opt/displaylab-socket/.env
sudo chmod 755 /opt/displaylab-socket/displaylab-socket

sudo systemctl daemon-reload
sudo systemctl enable --now displaylab-socket
sudo systemctl status displaylab-socket --no-pager
```

Check:

```bash
curl -s http://127.0.0.1:8090/health
journalctl -u displaylab-socket -f
```

---

## 5) Nginx reverse proxy for WebSocket (public)

Devices need a public WSS URL. Example subdomain: `socket.displaylab.net`

```bash
sudo tee /etc/nginx/sites-available/displaylab-socket >/dev/null <<'EOF'
server {
    listen 80;
    server_name socket.displaylab.net;

    location /health {
        proxy_pass http://127.0.0.1:8090/health;
    }

    location /ws {
        proxy_pass http://127.0.0.1:8090/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }

    # Do NOT expose /internal publicly.
    location /internal/ {
        deny all;
    }
}
EOF

sudo ln -sf /etc/nginx/sites-available/displaylab-socket /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

TLS (Let’s Encrypt):

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d socket.displaylab.net
```

Flutter/device connects to:

```text
wss://socket.displaylab.net/ws?device_token=<fcm_token>
```

---

## 6) Point DisplayLab PHP at the local Go service

On the same VPS, in DisplayLab `application/config/config.php`:

```php
$config['socket_notify_url'] = 'http://127.0.0.1:8090/internal/v1/notify';
$config['socket_notify_key'] = 'CHANGE_ME_TO_A_LONG_RANDOM_SECRET'; // same as .env
```

If PHP and Go are on **different** servers, use a private network URL and keep `/internal` firewalled. Do not expose `/internal` to the public internet.

Also update the same key defaults if you still have hardcoded fallbacks in `pushNotification.php` copies.

---

## 7) Firewall

```bash
# allow SSH + HTTP/HTTPS only
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
# do NOT open 8090 publicly
```

---

## 8) Smoke test on VPS

```bash
# health
curl -s http://127.0.0.1:8090/health

# notify (should return delivered:0 if no device connected)
curl -s -X POST http://127.0.0.1:8090/internal/v1/notify \
  -H 'Content-Type: application/json' \
  -H 'X-Internal-Key: CHANGE_ME_TO_A_LONG_RANDOM_SECRET' \
  -d '{"token":"test","title":"Display Lab","body":"Playlist Updated","data":{"playlist_update":"true"}}'

# from outside after TLS
curl -s https://socket.displaylab.net/health
```

Trigger a real playlist send from DisplayLab CMS and watch:

```bash
journalctl -u displaylab-socket -f
```

---

## Updates / redeploy

```bash
# upload new binary then:
sudo systemctl restart displaylab-socket
sudo systemctl status displaylab-socket --no-pager
```

---

## Production checklist

- [ ] Strong unique `INTERNAL_API_KEY`
- [ ] Go bound to `127.0.0.1:8090`
- [ ] `/internal` not public
- [ ] `wss://` TLS for devices
- [ ] PHP `socket_notify_url` + key match
- [ ] systemd enabled
- [ ] logs monitored (`journalctl -u displaylab-socket`)
