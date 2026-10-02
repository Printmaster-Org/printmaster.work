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

Health endpoint: `/healthz`.

## Site content

- Windows MSI downloads link to the stable GitHub Releases page; installers are versioned release assets.
- Debian/Ubuntu and Fedora/RHEL installation commands point to the official package endpoints.
- `https://docs.printmaster.work/` is the planned separate documentation site.
- Product claims are intentionally limited to currently implemented Agent and Server behavior; printer data availability varies with device and SNMP configuration.