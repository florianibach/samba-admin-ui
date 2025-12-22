Ich würde die UI so aufbauen, dass sie 80% über geführte Formulare abdeckt, aber immer einen “Escape Hatch” lässt: Config direkt bearbeiten (und zwar möglichst ohne dass die UI dir Dinge kaputt “zurückschreibt”).

Grundprinzip

UI-managed Shares/User leben in shares.d/ + users.d/ (oder managed/).

Custom/Manual Konfig bleibt unangetastet (z.B. in custom/ oder direkt in smb.conf).

Die UI liest alles, aber schreibt nur in ihren Bereich.


So kannst du jederzeit “besondere” Einträge hinzufügen, ohne dass die UI sie überschreibt.


---

Vorschlag Seitenstruktur

1) Dashboard

Ziel: Schnellzustand + letzte Aktionen

Status: smbd läuft? (Container: smbd -V + Prozesscheck)

“Config OK?” (Button: testparm)

Letzte Änderungen / Reload-Zeitpunkt

Warnungen:

„Share-Pfad existiert nicht“

„UID/GID mismatch“

„valid users verweist auf unbekannten User“


Quick actions:

Reload Samba

Validate config

Create Share (Wizard)




---

2) Shares

Ziel: Hauptarbeitsfläche

Liste (Table/Card)

Name, Pfad, RW/RO, Sichtbarkeit (browseable), “Managed/Manual”

“Allowed”: Users/Groups (Kurzform)

Health-Indicator (grün/gelb/rot) z.B. testparm/Path


Share-Detailseite

Tabs:

Settings (Form): Name, Path, RW/RO, browseable, guest, valid users, veto files (optional)

Access: Users/Groups per Multi-Select

Advanced (Raw): zeigt die effektive Share-Section als INI/Text (read-only oder editierbar nur wenn “Custom”)

Filesystem: Permissions-Check (owner/group/mode), “Fix permissions” (optional)


Aktionen:

Save → schreibt nur managed config

Clone share

Disable share (statt löschen)



---

3) Users & Groups

Ziel: Samba-User + Mapping sichtbar

Users

Liste: vater, mutter

Status: enabled/disabled, password set? (nicht das Passwort natürlich)

Aktionen:

Add Samba User (nur wenn Linux-User existiert)

Set/Reset Password

Enable/Disable

Remove Samba User



Groups

Liste: Gruppen aus /etc/group (bzw. Container)

“Members” anzeigen

Quick: “Add user to group” (optional)


💡 Wichtig: du kannst klar trennen:

Linux identity (name/uid/gid/groups) – read-only oder minimal

Samba account (passdb, enabled, password) – verwaltbar



---

4) Filesystem

Ziel: “Ordner anlegen + Rechte prüfen” (dein Flow)

Root mount auswählen (z.B. /shares)

Directory browser (nur innerhalb der gemounteten Roots)

Aktionen:

Create directory

Set owner/group (Dropdown user/group)

Apply preset permissions:

“Private user folder” (u:rwx g:--- o:---)

“Shared group folder” (2770 + setgid)



Anzeigen:

owner/group/mode

optional ACLs (getfacl) wenn du willst




---

5) Config Editor

Das ist dein “ich will trotzdem alles können”-Feature.

Aufbau als 2 Bereiche:

Managed config (read-only): zeigt, was die UI generiert (damit man versteht, warum etwas so ist)

Custom config (editable): für Dinge, die UI nicht abbildet


Konkret:

smb.conf (meist read-only, weil global riskant)

shares.d/managed/*.conf (read-only, von UI verwaltet)

shares.d/custom/*.conf (editable)

optional: global.d/custom.conf (für extra globals)


Features:

Syntax Highlight (ini)

“Validate” Button (testparm)

Diff vor dem Speichern

“Reload after save” Toggle


Wichtig: Beim Speichern nur in “custom” erlauben (oder mit Warnung), sonst überschreibt man sich selbst.


---

6) Services & Logs

Ziel: Debug ohne SSH

Button: reload (smbcontrol all reload-config)

Button: restart (wenn du es erlauben willst)

Logs:

log.smbd / stdout vom Container

Filter: errors/warnings


Active connections:

smbstatus (super hilfreich!)

offene Dateien, Sessions, Locks




---

7) Settings

Paths:

Config root: /etc/samba

Managed dir: /etc/samba/shares.d/managed

Custom dir: /etc/samba/shares.d/custom

Share roots: /shares (whitelist)


Defaults:

create mask / directory mask

browseable

audit options


Security:

UI auth (lokal)

Allowed subnets


Backup/Export:

Download config bundle (zip)

Import bundle




---

Der “Wizard” für deinen häufigsten Ablauf

Create Share Wizard (3 Schritte, super schnell)

1. Select folder



Create new: /shares/vater

Or pick existing


2. Access



Private (single user)

Group shared (select group)

RO/RW


3. Review



zeigt:

die INI-section

permissions, die gesetzt werden


Apply + Reload


Das ist genau der “mal eben freigeben”-Flow.


---

Was mir bei “UI + Raw Config” wichtig wäre

Damit du nie Angst hast, dass die UI dir was kaputt macht:

UI schreibt nur in managed/

Alles was nicht UI-supported ist → gehört in custom/

UI zeigt “effective config” (zusammengeführt), aber überschreibt nicht


Optional nice:

UI importiert existierende Shares als “Manual” (read-only), mit Button “Convert to managed” (mit Diff & Bestätigung)



---

Wenn du willst, kann ich dir daraus direkt:

eine Seitenliste als Sidebar-Navigation

die Datenstruktur (Share/User/ConfigFile)

und einen v0.1 Scope (damit du schnell ein MVP hast) runterschreiben.
