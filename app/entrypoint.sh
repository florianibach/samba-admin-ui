#!/usr/bin/env bash
set -euo pipefail

SMB_CONF="${SMB_CONF:-/etc/samba/smb.conf}"
HTTP_ADDR="${HTTP_ADDR:-:8080}"

# Create a minimal smb.conf if none exists yet
if [[ ! -f "$SMB_CONF" ]]; then
  mkdir -p "$(dirname "$SMB_CONF")"
  cat > "$SMB_CONF" <<'EOF'
[global]
  workgroup = WORKGROUP
  server role = standalone server
  map to guest = never
  log file = /var/log/samba/log.%m
  logging = file
  max log size = 1000
  smb ports = 445 139
  disable netbios = no
  server min protocol = SMB2
  include = /etc/samba/shares.d/ui/shares.conf
EOF
fi

# Also ensure samba lib exists (volume is mounted)
mkdir -p /var/lib/samba/private /var/lib/samba/printers
chmod 0700 /var/lib/samba/private


# Start Samba daemons in background
# (This is MVP-grade; later you may want s6/supervisor for robustness.)
mkdir -p /run/samba /var/log/samba
smbd -F --no-process-group &
nmbd -F --no-process-group &

# Start UI (foreground)
exec /app/samba-admin-ui
