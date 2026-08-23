# archivyy

## Setup

```sh
cp .env.example .env
openssl rand -base64 32   # paste into JWT_SECRET
```

## Run

With Docker (app + Postgres):

```sh
docker compose up --build
```

Locally, with Postgres in Docker:

```sh
docker compose up -d db
make run
```

Either way the app is at http://localhost:8080.

## Other commands

```sh
make          # list all targets
make build    # compile to bin/
make check    # fmt, vet, test
```
