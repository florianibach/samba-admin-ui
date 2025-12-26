# samba-admin-ui

A simple web UI to manage Samba users and shares for homelab environments.

This project is intentionally minimal and opinionated: it focuses on the most common Samba tasks without trying to replace full system configuration or enterprise tooling.

---
This project is built and maintained in my free time.  
If it helps you or saves you some time, you can support my work on [![BuyMeACoffee](https://raw.githubusercontent.com/pachadotdev/buymeacoffee-badges/main/bmc-black.svg)](https://buymeacoffee.com/floibach)

Thank you for your support!


## ✨ Features

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

---

## 🚫 Non-Goals

This project deliberately does **not** aim to:
- Replace full Samba configuration management
- Edit advanced Samba options
- Manage Linux groups or ACLs via UI (planned post-MVP)
- Be an enterprise-ready or multi-tenant solution

This is a **homelab-focused tool**.

---

## 🐳 Docker Usage

### Minimal `docker-compose.yml`

```yaml
services:
  samba-admin-ui:
    image: ghcr.io/florianibach/samba-admin-ui:latest
    container_name: samba-admin-ui
    ports:
      - "8080:8080"
    volumes:
      # Samba config (read-only)
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

## ⚠️ Important Notes

* The container runs as **root** to manage Samba and Linux users.
* Linux users are created without passwords and with `nologin`.
* Only users with UID ≥ 1000 are shown in the Linux users overview.
* This tool assumes you know what you are doing — it is designed for trusted environments.

---

## 📦 Data Persistence

You should persist at least:

* `/var/lib/samba` – Samba users and passwords
* `/data/` – internal application state
* `/etc/samba/shares.d` – UI-managed shares

---

## 🧭 Project Status

**MVP – stable and usable**

Planned post-MVP features:

* Linux group management
* User-to-group assignments
* Debug / desired-vs-actual view
* Improved responsive UI

---

## 📝 License

MIT

```

---

## ✅ Was du jetzt noch tun kannst (optional, aber empfehlenswert)

- Version taggen (`v0.1.0`)
- GitHub Release erstellen
- Docker Image pushen (`latest` + Version)

👉 Danach hast du:
- ein **echtes MVP**
- ein Projekt, das man guten Gewissens teilen kann
- eine stabile Basis für Version 1.0

Wenn du willst, helfe ich dir im nächsten Schritt gern noch bei:
- GitHub Action für Multi-Arch Docker Builds  
- Versioning-Strategie (`v0.x` → `v1.0`)  

Aber für den Moment: **Glückwunsch, MVP abgeschlossen** 👏
```
