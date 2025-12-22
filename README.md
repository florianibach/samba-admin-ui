# samba-gui


Ja – **den Ordner kannst du aus dem Container heraus anlegen**, *wenn* du dem Container die nötigen Rechte gibst. Und ja: im Container brauchst du **UID/GID + Namen** (mindestens für Gruppen-/Userauflösung).

## Ordner im Container anlegen: geht das?

**Ja**, mit diesem Setup:

* Host bind-mountet z.B. nur:

  * `/srv/disk0` → `/shares`
* Container läuft als **root** (oder mit passenden Capabilities)
* Dann kann der Container:

  * `mkdir /shares/vater`
  * `chown 1002:2000 /shares/vater` (oder `chown vater:eltern …` wenn Namen existieren)
  * `chmod …`

### Aber: was bleibt trotzdem Host-Arbeit?

Ganz ohne Host geht’s nicht, weil du mindestens einmal sicherstellen musst:

* `/srv/disk0` existiert und ist gemountet
* Ownership/Mode von `/srv/disk0` erlaubt dem Container-root das Anlegen (meist ja)
* ggf. Filesystem-Optionen passen (z.B. bei NFS/NTFS kann `chown` eingeschränkt sein)

Wenn `/srv/disk0` ein normales ext4/xfs/zfs ist: **kein Problem**.

---

## Sicherheits-Real Talk

Wenn du dem Container Root auf einen Host-Pfad gibst, ist das effektiv:

> **Root auf diesen Teil des Hosts.**

Für dein „Eltern-Share zu Hause“ ist das oft ok, aber ich würde es trotzdem sauber einkreisen:

* mount **nur** den Samba-Bereich, nicht `/`
* Container nicht nach außen exposen
* GUI mit Passwort / ggf. nur LAN
* Backups der Samba-Config

---

## Brauchst du im Container wirklich den Linux-User *mit Name*?

Du hast es fast perfekt zusammengefasst:

### Für Dateizugriff (mkdir/chown) reicht technisch:

✅ **UID/GID** (numerisch), weil Linux am Ende numerisch entscheidet.

Du könntest also im Container auch:

```bash
chown 1002:2000 /shares/vater
```

ohne dass `vater` als Name existiert.

### Für Samba-Konfig ist der Name aber sehr hilfreich bzw. oft nötig:

* `valid users = vater` → Samba muss `vater` über NSS (getpwnam) auflösen
* `valid users = @eltern` → Gruppe muss auflösbar sein

👉 Deshalb: **ja**, im Container solltest du
✅ **User/Group mit denselben UID/GID und denselben Namen anlegen**
(selbst wenn du rein für chown nur Zahlen bräuchtest).

---

## Minimales Modell für deinen Wunsch (“noch weniger Host”)

### Host macht nur:

1. `/srv/disk0` mounten/erstellen
2. Container starten

### Container macht:

* Linux-User/Group *im Container* anlegen (mit UID/GID)
* Verzeichnisse anlegen
* chown/chmod setzen
* Samba-User per `smbpasswd` verwalten
* Shares konfigurieren & reloaden

**Haken:** Woher kennt der Container UID/GID?

* entweder du gibst UID/GID in der UI ein (meh)
* oder du “verwaltetst” die IDs komplett im Container und hörst auf, Host-User anzulegen
* oder du mountest `/etc/passwd` und `/etc/group` read-only rein (geht, aber ich mag’s nicht)

Für dein ursprüngliches Ziel (“Host-User anlegen, fertig”) ist am saubersten:

* UID/GID sind bekannt (weil du sie beim Host-User anlegen siehst)
* du gibst sie einmal in eine Config/ENV/GUI
* Container legt die gleichen Identitäten intern an

---

## Empfehlung (praktisch & wartbar)

Wenn du wirklich **Host minimal** willst, aber ohne UID-Gefrickel:

### Variante A (mein Favorit)

* Host: Users **einmalig** anlegen (wie du es geplant hast)
* Container: beim Start liest er eine kleine YAML/JSON mit Users:

  ```yaml
  users:
    - name: vater
      uid: 1002
      groups: [eltern]
    - name: mutter
      uid: 1003
      groups: [eltern]
  groupIds:
    eltern: 2000
  ```
* Container erzeugt daraus `useradd/groupadd`, legt Ordner an, setzt Rechte.

➡️ Du musst nie `/etc/passwd` mounten und hast trotzdem „nur einmal“ Host-Arbeit.

---

## Deine letzte Frage, klar beantwortet

> „im container brauche ich uid/gid, sowie linux user name?“

✅ **Ja**, wenn du `valid users = vater` oder `@eltern` nutzen willst.

* **UID/GID**: damit Filesystemrechte stimmen
* **Name**: damit Samba User/Group auflösen kann

---

Wenn du magst, skizziere ich dir als nächstes eine konkrete Compose + Init-Logik (ohne viel Overengineering), die:

* Users/Groups aus einer Datei anlegt
* Ordner erzeugt
* Samba shares aus `shares.d/*.conf` lädt
* und `smbcontrol all reload-config` nach UI-Änderungen macht.
