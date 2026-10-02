# PrintMaster Website

Public home for PrintMaster, served by a small Go HTTP server with static assets embedded in the binary. No JavaScript framework or runtime dependency required.

## Run locally

```sh
go run .
```

Open <http://localhost:8080>. Set `PORT` to change the listen port.

## Build and run with Docker

```sh
docker build -t printmaster-site .
docker run --rm -p 8080:8080 printmaster-site
```

Every push to `main` runs tests and publishes the image to GitHub Container Registry:

```sh
docker pull ghcr.io/printmaster-org/printmaster.work:main
docker run -d --name printmaster-site --restart unless-stopped -p 8080:8080 ghcr.io/printmaster-org/printmaster.work:main
```

The workflow also publishes an immutable `sha-<commit>` tag. Set the GHCR package to **public** in GitHub package settings for hosts that pull without registry credentials; package visibility is not changed automatically by the workflow. Pull the updated `main` image and recreate the container to deploy a newer build.

Health endpoint: `/healthz`.

## Release catalog updates

The site reads public GitHub Release metadata once at startup and serves an in-memory catalog. It never downloads or proxies release binaries. After an Agent or Server stable/beta release is published, the PrintMaster release workflow sends a signed notification to `POST /api/releases/refresh`; the site refreshes metadata and the downloads page links directly to GitHub assets. Duplicate notifications for a release already in the catalog are ignored.

To enable automatic refresh:

1. Generate one shared random secret (for example, with `openssl rand -hex 32`).
2. Add it to the PrintMaster repo's Actions secrets as `PRINTMASTER_WEBSITE_RELEASE_SECRET`.
3. Pass the same value to the website container as `RELEASE_WEBHOOK_SECRET`. For example, add `RELEASE_WEBHOOK_SECRET: ${RELEASE_WEBHOOK_SECRET}` under the service's `environment:` block, then store the value in the host's `.env` file.

The workflow skips notification if its secret is unset, and notification failures do not fail a product release. Restarting the website performs a fresh catalog read; if startup refresh fails, the first downloads-page API request retries. No recurring GitHub polling is configured. Optionally set `GITHUB_TOKEN` on the website container to a read-only GitHub token for higher API rate limits.

## CI and dependency updates

- Pull requests run Go tests, `go vet`, and a Docker build without publishing.
- Pushes to `main` run Go tests, then publish the image to GHCR.
- Dependabot checks Go modules, Docker base images, and GitHub Actions weekly.

## Site content

- Windows MSI downloads link directly to versioned GitHub Release assets from `/downloads`.
- Debian/Ubuntu and Fedora/RHEL installation commands point to the official package endpoints.
- `https://docs.printmaster.work/` is the planned separate documentation site.
- Product claims are limited to implemented Agent and Server behavior; printer data availability varies with device and SNMP configuration.
