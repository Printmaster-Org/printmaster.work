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

## CI and dependency updates

- Pull requests run Go tests, `go vet`, and a Docker build without publishing.
- Pushes to `main` run Go tests, then publish the image to GHCR.
- Dependabot checks Go modules, Docker base images, and GitHub Actions weekly.

## Site content

- Windows MSI downloads link to the stable GitHub Releases page; installers are versioned release assets.
- Debian/Ubuntu and Fedora/RHEL installation commands point to the official package endpoints.
- `https://docs.printmaster.work/` is the planned separate documentation site.
- Product claims are limited to implemented Agent and Server behavior; printer data availability varies with device and SNMP configuration.
