FROM golang:1.25.5-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/samba-admin-ui ./main.go

FROM debian:bookworm-slim

# Samba utils: testparm, pdbedit, smbcontrol, smbd/nmbd
RUN apt-get update \
  && apt-get install -y --no-install-recommends \
     samba \
     smbclient \
     ca-certificates \
     tini \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /out/samba-admin-ui /app/samba-admin-ui
COPY templates /app/templates
COPY static /app/static
COPY entrypoint.sh /app/entrypoint.sh

RUN chmod +x /app/entrypoint.sh \
  && mkdir -p /shares /var/log/samba /run/samba \
  && mkdir -p /etc/samba

EXPOSE 8080 445 139 137/udp 138/udp

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["/app/entrypoint.sh"]
