
# Archivyy

**Try it out:** https://pranav.hackclub.app

Archivyy is a self-hosted file archiving and streaming platform built with **Go, Echo, HTMX, and S3-compatible storage**.

### Features

* File uploads to S3/SeaweedFS
* JWT authentication
* SQLite/PostgreSQL support
* Metadata stored locally in JSON for faster archive loading
* Custom file titles
* Custom preview thumbnails for video and audio files
* Dedicated file streaming through `/stream/:filename`
* HTTP Range support for media streaming
* Lightweight frontend built with **CrossCode**

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

### Architecture

```text
Browser
   ↓
Go / Echo
   ↓
meta.json → archive metadata & previews
   ↓
S3 / SeaweedFS → actual files
   ↓
/stream/:filename → file streaming
```

Archive metadata is loaded separately from the actual files, so the homepage doesn't need to download every archived object just to display the file cards.

### Credits

Frontend built with **CrossCode**.

Thanks to my friend **Mitul** for helping with the initial SQL authentication implementation.
