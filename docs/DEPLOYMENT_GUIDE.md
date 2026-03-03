# DOGMud Deployment Guide

Deploy DOGMud to a DigitalOcean Droplet with Docker and Caddy for
automatic HTTPS. No prior Docker or server experience required.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Prerequisites](#2-prerequisites)
3. [Create the Droplet](#3-create-the-droplet)
4. [Initial Server Setup](#4-initial-server-setup)
5. [Install Docker](#5-install-docker)
6. [Clone and Configure](#6-clone-and-configure)
7. [Set Up Docker Compose with Caddy](#7-set-up-docker-compose-with-caddy)
8. [Domain & DNS](#8-domain--dns)
9. [First Deploy](#9-first-deploy)
10. [Maintenance](#10-maintenance)

---

## 1. Overview

### What We're Deploying

- **MUD server** — the Go binary that runs the game world
- **Web portal** — browser-based client served over HTTPS (port 443)
- **Telnet** — classic MUD client access (port 33333)

### The Stack

| Component      | Role                                          |
|----------------|-----------------------------------------------|
| DigitalOcean   | Cloud server (Droplet)                        |
| Ubuntu 24.04   | Operating system                              |
| Docker         | Runs the MUD in an isolated container         |
| Caddy          | Reverse proxy with automatic HTTPS via        |
|                | Let's Encrypt                                 |

### Cost Estimate

| Item                   | Cost          |
|------------------------|---------------|
| Droplet (1GB/1vCPU)   | ~$6/month     |
| Domain name            | ~$12/year     |
| **Total**              | ~$7/month     |

The $6 Droplet (1 GB RAM) is enough for launch. If you outgrow
it, DigitalOcean lets you resize to 2 GB ($12/mo) in about 60
seconds via their UI with no data loss.

---

## 2. Prerequisites

Before you start, you need:

1. **A DigitalOcean account** — sign up at
   [digitalocean.com](https://www.digitalocean.com/)
2. **A domain name** — e.g. `dogmud.com` (see Section 8)
3. **Access to the DOGMud GitHub repo** — you'll clone it onto
   the server
4. **A terminal on your local machine** — PowerShell (Windows),
   Terminal (Mac), or any Linux terminal

---

## 3. Create the Droplet

### 3.1 Generate an SSH Key (Windows)

If you already have an SSH key, skip to 3.2.

Open PowerShell and run:

```powershell
ssh-keygen -t ed25519 -C "your-email@example.com"
```

Press Enter to accept the default file location. Set a passphrase
or press Enter for none. Then copy your public key:

```powershell
cat ~/.ssh/id_ed25519.pub
```

Copy the entire output — you'll paste it into DigitalOcean next.

### 3.2 Create the Droplet in DigitalOcean

1. Log in to [cloud.digitalocean.com](https://cloud.digitalocean.com)
2. Click **Create** → **Droplets**
3. **Region**: Choose the datacenter closest to your players
   (e.g., New York, San Francisco, London, Frankfurt)
4. **Image**: Select **Ubuntu** → **24.04 (LTS) x64**
5. **Size**: Click **Basic** → **Regular** → select the
   **$6/mo** plan (1 GB RAM / 1 vCPU / 25 GB SSD)
6. **Authentication**: Select **SSH Key** → click **New SSH Key**
   → paste your public key from step 3.1 → give it a name →
   click **Add SSH Key**
7. **Hostname**: Enter something memorable like `dogmud`
8. Click **Create Droplet**

Wait ~60 seconds. Note the **IP address** shown on your Droplets
page — you'll need it throughout this guide. We'll call it
`YOUR_SERVER_IP`.

---

## 4. Initial Server Setup

### 4.1 SSH Into Your Server

```bash
ssh root@YOUR_SERVER_IP
```

Type `yes` if prompted about the host fingerprint.

### 4.2 Create a Non-Root User

Running everything as root is risky. Create a regular user:

```bash
adduser mudadmin
```

Set a password when prompted. Press Enter through the other fields.

Give the user sudo access:

```bash
usermod -aG sudo mudadmin
```

Copy your SSH key to the new user so you can log in as them:

```bash
rsync --archive --chown=mudadmin:mudadmin ~/.ssh /home/mudadmin
```

### 4.3 Set Up the Firewall

```bash
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP (Caddy needs this for HTTPS challenges)
ufw allow 443/tcp   # HTTPS (web portal)
ufw allow 33333/tcp # Telnet (MUD clients)
ufw enable
```

Type `y` when prompted.

### 4.4 Switch to the New User

Log out of root:

```bash
exit
```

Log back in as your new user:

```bash
ssh mudadmin@YOUR_SERVER_IP
```

From now on, always use `mudadmin` — never `root`.

### 4.5 Enable Swap (Required for 1 GB Droplet)

The Go compiler needs more than 1 GB to build. Adding swap
prevents out-of-memory kills during `docker build`:

```bash
sudo fallocate -l 1G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

Verify it's active:

```bash
free -h
```

You should see `Swap: 1.0G` in the output.

---

## 5. Install Docker

These commands install Docker from the official repository on
Ubuntu 24.04.

### 5.1 Add Docker's GPG Key and Repository

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
  -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

echo \
  "deb [arch=$(dpkg --print-architecture) \
  signed-by=/etc/apt/keyrings/docker.asc] \
  https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "${VERSION_CODENAME}") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
```

### 5.2 Install Docker Engine and Compose Plugin

```bash
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli \
  containerd.io docker-buildx-plugin docker-compose-plugin
```

### 5.3 Add Your User to the Docker Group

This lets you run `docker` commands without `sudo`:

```bash
sudo usermod -aG docker mudadmin
```

**You must log out and back in** for the group change to take
effect. The `exit` command below will disconnect you from the
server — you'll be back in your local terminal. Then SSH in
again:

```bash
exit
```

**>>> You are now back on your LOCAL machine. SSH in again: <<<**

```bash
ssh mudadmin@YOUR_SERVER_IP
```

### 5.4 Verify Installation

```bash
docker --version
docker compose version
```

Both commands should print version numbers without errors.

---

## 6. Clone and Configure

### 6.1 Clone the Repository

```bash
cd ~
git clone https://github.com/pruuk/DOGMud.git
cd DOGMud
```

Create required folders that aren't in the repo (the engine
validates these on startup):

```bash
mkdir -p _datafiles/world/dogmud/plugin-data
mkdir -p _datafiles/world/dogmud/users
mkdir -p _datafiles/world/dogmud/rooms.instances
mkdir -p _datafiles/world/dogmud/mobs.instances
```

### 6.2 Create a Production Config Override

The MUD reads `_datafiles/config.yaml` as its base config, then
merges in an override file specified by the `CONFIG_PATH`
environment variable. This lets you customize production settings
without modifying the tracked config file.

Create the config directory:

```bash
mkdir -p ~/mud-config
```

Then open a new file in nano:

```bash
nano ~/mud-config/config-production.yaml
```

Paste the following contents into nano (right-click to paste in
most terminals):

```yaml
# Production config overrides for DOGMud
# Only include values you want to change from the defaults
# in _datafiles/config.yaml

FilePaths:
  # Set this to your actual domain (no http:// prefix)
  WebDomain: "yourdomain.com"

Network:
  # Disable HTTPS on the MUD itself — Caddy handles TLS termination
  HttpsPort: 0
  # The MUD listens on port 80 internally; Caddy proxies to it
  HttpPort: 80
  # Telnet ports
  TelnetPort: [33333]
  # Player limits tuned for 1GB RAM
  # If you upgrade to 2GB, you can raise these
  MaxTelnetConnections: 80
  MaxHumanConnections: 50
  MaxAIConnections: 10
```

Save and exit nano: press `Ctrl+O`, then `Enter`, then `Ctrl+X`.

**Important:** Replace `yourdomain.com` with your actual domain
name (see Section 8).

---

## 7. Set Up Docker Compose with Caddy

### 7.1 Create the Production Compose File

Create a new compose file for production. This replaces the
development `compose.yml` — we won't modify the tracked file.

```bash
cat > ~/DOGMud/compose.production.yml << 'EOF'
services:
  server:
    container_name: "dogmud-server"
    build:
      context: .
      dockerfile: ./provisioning/Dockerfile
      args:
        - BIN=go-mud-server
    environment:
      SERVICE_NAME: go-mud-server
      PORT: 33333
      CONFIG_PATH: /mud-config/config-production.yaml
    ports:
      # Telnet — exposed directly to the internet
      - "33333:33333"
    volumes:
      # Persist game data across container rebuilds
      - ./_datafiles:/app/_datafiles
      # Mount the production config override
      - /home/mudadmin/mud-config:/mud-config:ro
    networks:
      - mud_network
    restart: unless-stopped

  caddy:
    container_name: "dogmud-caddy"
    image: caddy:2-alpine
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"  # HTTP/3 (QUIC)
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data      # TLS certificates
      - caddy_config:/config  # Caddy config cache
    networks:
      - mud_network
    restart: unless-stopped
    depends_on:
      - server

volumes:
  caddy_data:
  caddy_config:

networks:
  mud_network:
    name: mud_network
EOF
```

### Key Differences from Development compose.yml

| Change                          | Why                              |
|---------------------------------|----------------------------------|
| `_datafiles` bind mount         | Persist player data, room/mob    |
|                                 | instances across rebuilds        |
| `CONFIG_PATH` env var           | Use production config without    |
|                                 | modifying tracked files          |
| `restart: unless-stopped`       | Auto-restart on crash or reboot  |
| Caddy service added             | Automatic HTTPS + reverse proxy  |
| busybox log viewer removed      | Use `docker compose logs` instead|
| terminal service removed        | Not needed in production         |

### 7.2 Create the Caddyfile

```bash
cat > ~/DOGMud/Caddyfile << 'EOF'
yourdomain.com {
    reverse_proxy server:80
}
EOF
```

**Important:** Replace `yourdomain.com` with your actual domain.

That's the entire Caddyfile. Caddy automatically:
- Obtains a Let's Encrypt TLS certificate for your domain
- Renews it before expiry
- Redirects HTTP to HTTPS
- Serves your web portal over HTTPS

---

## 8. Domain & DNS

### 8.1 Register a Domain

If you don't have a domain yet, register one at
[Cloudflare Registrar](https://www.cloudflare.com/products/registrar/)
— at-cost pricing (~$10–12/year for `.com`) and built-in DNS
management.

### 8.2 Point Your Domain to Your Droplet (Cloudflare)

1. Log in to [dash.cloudflare.com](https://dash.cloudflare.com)
2. Click on your domain
3. Click **DNS** → **Records** in the left sidebar
4. Click **Add Record** and fill in:

| Setting        | Value            |
|----------------|------------------|
| Type           | `A`              |
| Name           | `@`              |
| IPv4 address   | `YOUR_SERVER_IP` |
| Proxy status   | **DNS only** (grey cloud — see below) |
| TTL            | Auto             |

5. Click **Save**

**Important: Set proxy status to "DNS only" (grey cloud).**
Click the orange cloud icon to toggle it off. If Cloudflare's
proxy is enabled, it will intercept HTTPS traffic and prevent
Caddy from obtaining its Let's Encrypt certificate.

If you also want `www.yourdomain.com` to work, add a second
record the same way but with Name set to `www` (also DNS only).
Then update your Caddyfile to handle both:

```
yourdomain.com, www.yourdomain.com {
    reverse_proxy server:80
}
```

### 8.3 Wait for DNS Propagation

Cloudflare DNS is fast — changes usually take effect within 1–5
minutes. You can verify from your server:

```bash
dig +short yourdomain.com
```

It should return your Droplet's IP. You can also check at
[dnschecker.org](https://dnschecker.org/).

Verify from your server:

```bash
dig +short yourdomain.com
```

It should return your Droplet's IP address.

---

## 9. First Deploy

### 9.1 Build and Start

```bash
cd ~/DOGMud
docker compose -f compose.production.yml up -d --build
```

This will:
1. Build the Go binary inside a Docker container
2. Start the MUD server
3. Start Caddy (which will automatically get your TLS cert)

First build takes 2–5 minutes. Subsequent rebuilds are faster
thanks to Docker layer caching.

### 9.2 Watch the Logs

```bash
docker compose -f compose.production.yml logs -f
```

You should see:
- The MUD server starting up and loading zones/mobs/items
- Caddy obtaining a TLS certificate for your domain

Press `Ctrl+C` to stop watching logs (the server keeps running).

### 9.3 Test Everything

**Test telnet:**

From your local machine, use any MUD client (Mudlet, TinTin++,
PuTTY in raw mode) to connect to:

```
Host: yourdomain.com
Port: 33333
```

Or from a terminal:

```bash
telnet yourdomain.com 33333
```

**Test the web portal:**

Open a browser and go to:

```
https://yourdomain.com
```

You should see the DOGMud web client with a valid HTTPS
certificate (lock icon in the address bar).

---

## 10. Maintenance

### 10.1 Updating Code (Deploy Workflow)

When you push code changes to GitHub and want to deploy them:

```bash
cd ~/DOGMud
git pull origin master
docker compose -f compose.production.yml up -d --build
```

That's the entire deploy workflow:
1. `git pull` fetches the latest code
2. `up -d --build` rebuilds the binary and restarts the container

The server will be briefly unavailable during the rebuild
(typically 30–60 seconds). Players connected via telnet will be
disconnected.

### 10.2 Viewing Logs

**Follow logs in real-time:**

```bash
docker compose -f compose.production.yml logs -f server
```

**View the last 100 lines:**

```bash
docker compose -f compose.production.yml logs --tail 100 server
```

**View Caddy logs:**

```bash
docker compose -f compose.production.yml logs caddy
```

### 10.3 Restarting the Server

**Restart just the MUD server (keeps Caddy running):**

```bash
docker compose -f compose.production.yml restart server
```

**Stop everything:**

```bash
docker compose -f compose.production.yml down
```

**Start everything back up:**

```bash
docker compose -f compose.production.yml up -d
```

### 10.4 Backing Up Game Data

The `_datafiles/` directory contains all runtime-mutable data
that you need to back up:

| Directory                              | Contains            |
|----------------------------------------|---------------------|
| `_datafiles/users/`                    | Player accounts     |
| `_datafiles/world/dogmud/rooms.instances/` | Room state      |
| `_datafiles/world/dogmud/mobs.instances/`  | Mob state       |

**Create a backup:**

```bash
BACKUP_DIR=~/backups/$(date +%Y-%m-%d_%H%M%S)
mkdir -p "$BACKUP_DIR"
cp -r ~/DOGMud/_datafiles/users "$BACKUP_DIR/"
cp -r ~/DOGMud/_datafiles/world/dogmud/rooms.instances \
  "$BACKUP_DIR/"
cp -r ~/DOGMud/_datafiles/world/dogmud/mobs.instances \
  "$BACKUP_DIR/"
echo "Backup saved to $BACKUP_DIR"
```

**Automate daily backups with cron:**

```bash
crontab -e
```

Add this line (backs up at 4:00 AM daily, keeps 30 days):

```
0 4 * * * BACKUP_DIR=~/backups/$(date +\%Y-\%m-\%d) && mkdir -p "$BACKUP_DIR" && cp -r ~/DOGMud/_datafiles/users "$BACKUP_DIR/" && cp -r ~/DOGMud/_datafiles/world/dogmud/rooms.instances "$BACKUP_DIR/" && cp -r ~/DOGMud/_datafiles/world/dogmud/mobs.instances "$BACKUP_DIR/" && find ~/backups -maxdepth 1 -mtime +30 -exec rm -rf {} \;
```

### 10.5 Crash Recovery

The `restart: unless-stopped` policy in the compose file means:

- If the MUD server crashes, Docker restarts it automatically
- If the Droplet reboots, Docker restarts all services on boot
- The only time it stays stopped is if you explicitly run
  `docker compose down`

To verify services are set to auto-start on boot:

```bash
sudo systemctl enable docker
```

### 10.6 Monitoring Disk and RAM

**Check RAM usage:**

```bash
free -h
```

**Check disk usage:**

```bash
df -h /
```

**Check container resource usage:**

```bash
docker stats --no-stream
```

If disk space is getting low, clean up old Docker images:

```bash
docker image prune -f
```

### 10.7 Upgrading the Droplet

If `docker stats` shows memory consistently above 80%, or you
want to support more players, upgrade to the $12/mo Droplet
(2 GB RAM):

1. Go to your Droplet page on DigitalOcean
2. Click **Resize** in the left sidebar
3. Select the $12/mo plan (2 GB / 1 vCPU / 50 GB)
4. Click **Resize** — takes about 60 seconds, no data loss
5. Update player limits in `~/mud-config/config-production.yaml`:

```yaml
  MaxTelnetConnections: 200
  MaxHumanConnections: 150
  MaxAIConnections: 30
```

6. Restart: `docker compose -f compose.production.yml restart server`

### 10.9 Updating the Server OS

Keep Ubuntu updated for security patches:

```bash
sudo apt update && sudo apt upgrade -y
```

Run this monthly or whenever you see a security advisory. If a
kernel update is applied, reboot:

```bash
sudo reboot
```

Docker and your MUD will restart automatically after the reboot
(thanks to the restart policy and `systemctl enable docker`).

### 10.10 Updating Docker

Docker updates come through the same package manager:

```bash
sudo apt update && sudo apt upgrade -y
```

This is safe — Docker upgrades don't affect your containers or
data.

---

## Quick Reference

| Task                     | Command                                    |
|--------------------------|--------------------------------------------|
| Deploy update            | `git pull && docker compose -f compose.production.yml up -d --build` |
| View logs                | `docker compose -f compose.production.yml logs -f server` |
| Restart server           | `docker compose -f compose.production.yml restart server` |
| Stop everything          | `docker compose -f compose.production.yml down` |
| Start everything         | `docker compose -f compose.production.yml up -d` |
| Backup data              | See Section 10.4                           |
| Check RAM/disk           | `free -h` / `df -h /`                     |
| OS updates               | `sudo apt update && sudo apt upgrade -y`   |

---

## Troubleshooting

### Caddy won't get a certificate

- Make sure your domain's A record points to your Droplet IP
- Make sure ports 80 and 443 are open: `sudo ufw status`
- Make sure the domain in your Caddyfile matches your actual domain
- Check Caddy logs: `docker compose -f compose.production.yml logs caddy`

### Can't connect via telnet

- Make sure port 33333 is open: `sudo ufw status`
- Check the server is running: `docker compose -f compose.production.yml ps`
- Check server logs for startup errors

### Web portal loads but shows "connection refused"

- The MUD server might still be starting up — wait 30 seconds
- Check that `HttpPort: 80` is set in your production config
- Check that `HttpsPort: 0` is set (Caddy handles HTTPS, not
  the MUD)

### Container keeps restarting

Check the logs for the crash reason:

```bash
docker compose -f compose.production.yml logs --tail 50 server
```

Common causes:
- Missing or malformed YAML in `_datafiles/`
- Port conflict (another service using port 33333 or 80)

### SSH connection refused after firewall setup

If you lock yourself out, use DigitalOcean's **Console** tab on
your Droplet page to access the server without SSH. Then fix the
firewall:

```bash
sudo ufw allow 22/tcp
sudo ufw reload
```
