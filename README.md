# samba-admin-ui

[![GitHub Repo](https://img.shields.io/badge/GitHub-Repository-blue?logo=github)](https://github.com/florianibach/samba-admin-ui)

[![DockerHub Repo](https://img.shields.io/badge/Docker_Hub-Repository-blue?logo=docker)](https://hub.docker.com/r/floibach/samba-admin-ui)


A simple web UI to manage Samba users and shares for homelab environments.

This project is intentionally minimal and opinionated: it focuses on the most common Samba tasks without trying to replace full system configuration or enterprise tooling.

---
This project is built and maintained in my free time.  
If it helps you or saves you some time, you can support my work on [![BuyMeACoffee](https://raw.githubusercontent.com/pachadotdev/buymeacoffee-badges/main/bmc-black.svg)](https://buymeacoffee.com/floibach)

Thank you for your support!


## Features

### Samba
- List Samba users
- Create Samba users (with password confirmation)
- Enable / disable Samba users
- Delete Samba users
- Create, enable, disable and delete Samba shares
- UI-managed shares are kept separate from manually managed shares

### Linux (read-only in UI)
- List Linux users (UID ≥ 1000)
- Show UID and group IDs
- Indicate whether a Samba user exists as a Linux user

### Architecture
- Runs fully containerized
- Uses SQLite for internal state
- Linux users are created automatically on container start if missing
- Samba configuration (`smb.conf`) is mounted read-only
- No direct editing of system files through the UI
- Runs on Raspberry Pi

---

## Non-Goals

This project deliberately does **not** aim to:
- Replace full Samba configuration management
- Edit advanced Samba options
- Manage Linux groups or ACLs via UI (planned post-MVP)
- Be an enterprise-ready or multi-tenant solution

This is a **homelab-focused tool**.

---

## Docker Usage

### Minimal `docker-compose.yml`

```yaml
services:
  samba-admin-ui:
    image: ghcr.io/florianibach/samba-admin-ui:latest
    container_name: samba-admin-ui
    ports:
      - "8080:8080"
    volumes:
      # (optional) Samba config (read-only)
      - ./samba-admin-ui/samba/smb.conf:/etc/samba/smb.conf:ro      

      # UI-managed share definitions
      - ./samba-admin-ui/samba/shares.d:/etc/samba/shares.d

      # Samba internal databases (users, passwords, state)
      - ./samba-admin-ui/samba-lib:/var/lib/samba

      # Internal app database (SQLite)
      - ./samba-admin-ui/data:/data

      # Actual share paths on the host
      - /srv/disk0:/shares
````

Then open:

```
http://localhost:8080
```

---

## Important Notes

* The container runs as **root** to manage Samba and Linux users.
* Linux users are created without passwords and with `nologin`.
* Only users with UID ≥ 1000 are shown in the Linux users overview.
* This tool assumes you know what you are doing — it is designed for trusted environments.

---

## Data Persistence

You should persist at least:

* `/var/lib/samba` – Samba users and passwords
* `/data/` – internal application state
* `/etc/samba/shares.d` – UI-managed shares

If you want to mount an existing samba configuration, mount (you can mount this as read-only):
* `/etc/samba/smb.conf` - must contain `include = /etc/samba/shares.d/ui/shares.conf` at the end of the global section


---

## Project Status

**MVP – stable and usable**

Planned post-MVP features:

* Linux group management
* User-to-group assignments
* Debug / desired-vs-actual view
* Improved responsive UI

---

## License

MIT

```
