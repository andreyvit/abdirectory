#!/bin/bash
set -euo pipefail
set -x

service="abdirectory"
username="andreyvit"

# -- deploy code ---------------------------------------------------------------
sudo install -d -m755 -g$username -o$username /srv/$service/bin
sudo install -d -m700 -g$username -o$username /srv/$service/secrets
sudo install -d -m755 -g$username -o$username /srv/$service/public

sudo install -m755 -g$username -o$username $service-linux-amd64 /srv/$service/bin/$service


# -- configure daemon ----------------------------------------------------------
sudo install -m644 -groot -oroot /dev/stdin /etc/systemd/system/$service.service <<EOF
[Unit]
Description=AB Directory Updater
After=network.target

[Service]
User=$username
Restart=always
PIDFile=/run/$service.pid
Type=simple
ExecStart=/srv/$service/bin/$service -o /srv/$service/public/index.html -d -secret /srv/$service/secrets/abdirectory_secret.json
KillMode=process
# EnvironmentFile=/srv/$service/$service.conf

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable $service


# -- configure Cadyy -----------------------------------------------------------
sudo install -m644 -groot -oroot /dev/stdin /srv/$service/Caddyfile <<EOF
ab.tarantsov.com {
    tls andrey@tarantsov.com
    root /srv/$service/public
    gzip
}
EOF

# -- start ---------------------------------------------------------------------
sudo systemctl restart $service
