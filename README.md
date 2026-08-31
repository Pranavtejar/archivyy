# Archivyy

**Try it out:** https://pranav.hackclub.app

Archivyy is a self-hosted file archiving and streaming platform built with **Go, Echo, HTMX, and S3-compatible storage**.

### Features

* File uploads to S3/SeaweedFS
* Direct file streaming
* JWT authentication
* SQLite/PostgreSQL support
* Lightweight HTMX frontend

### Run locally

```bash
git clone <repo-url>
cd archivyy
go mod tidy
```

Create `.env`:

```env
JWT_SECRET=your-secret
S3_ACCESS_KEY=admin
S3_SECRET_KEY=secret
S3_ENDPOINT=http://localhost:8333
S3_BUCKET=archive
PORT=8081
```

Start SeaweedFS, then:

```bash
go run main.go
```

Archivyy will be available at `http://localhost:8081`.

### Architecture

```text
Browser → Go/Echo → SeaweedFS/S3
                  ↘ Database
```

Files are streamed directly from storage instead of being loaded entirely into memory.

