#!/bin/sh
set -e

systemctl daemon-reload >/dev/null 2>&1 || true

echo "deepcoolgo: enable with  sudo systemctl enable --now deepcoolgo.service"
echo "deepcoolgo: config at    /etc/deepcoolgo/DCGO/config.yml"
