# cs-namecheap

A [Connectify](https://github.com/connectify-studio) connector for the
[Namecheap API](https://www.namecheap.com/support/api/intro/). It's a single,
self-contained command-line binary that talks to the Namecheap XML API and
prints results as a table, JSON, or CSV.

## Install

### Download a release (recommended)

Grab the archive for your platform from the
[Releases page](https://github.com/connectify-studio/cs-namecheap/releases/latest),
unpack it, and put the `cs-namecheap` binary somewhere on your `PATH`.

```sh
# Linux (amd64) example — adjust the version and platform to match the asset name.
curl -LO https://github.com/connectify-studio/cs-namecheap/releases/latest/download/cs-namecheap_0.1.0_linux_amd64.tar.gz
tar -xzf cs-namecheap_0.1.0_linux_amd64.tar.gz
sudo mv cs-namecheap /usr/local/bin/
```

Binaries are published for Linux, macOS, and Windows on both `amd64` and
`arm64`. Each release also includes a `checksums.txt` you can verify against.

### Install with Go

```sh
go install github.com/connectify-studio/cs-namecheap@latest
```

### Build from source

```sh
git clone https://github.com/connectify-studio/cs-namecheap
cd cs-namecheap
go build -o cs-namecheap .
```

Check the version of a build at any time:

```sh
cs-namecheap --version
```

## Prerequisites

Before the connector can reach Namecheap you must, in the
[Namecheap dashboard](https://ap.www.namecheap.com/settings/tools/apiaccess/):

1. **Enable API access** on your account.
2. **Whitelist the client IP** the requests will originate from. By default the
   connector auto-detects your public IP (via `https://api.ipify.org`) and
   caches it; override it with `--client-ip` if you route through a fixed
   address.

You'll need your **API user**, **API key**, and **Namecheap username**.

## Authentication

On first use you'll be prompted for your credentials, which are then stored
locally so you only enter them once. You can also set them up-front without the
prompt:

```sh
cs-namecheap config set api_user YOUR_API_USER
cs-namecheap config set api_key  YOUR_API_KEY
cs-namecheap config set username YOUR_USERNAME

cs-namecheap config list   # secrets are masked
cs-namecheap config get api_user
```

Only `api_user`, `api_key`, and `username` are valid keys; `config set` rejects
anything else.

## Usage

### List domains

```sh
cs-namecheap get-domains
```

By default this fetches **all** domains in the account, paging through the
Namecheap API as needed. Pass `--page` to retrieve a single page instead.

Options:

| Flag           | Description                                              |
| -------------- | -------------------------------------------------------- |
| `--page`       | Fetch only this page (1-based); default fetches all      |
| `--page-size`  | Results per page (max 100)                               |
| `--search`     | Filter by search term                                    |
| `--client-ip`  | Client IP to send (default: auto-detected public IP)     |
| `--sandbox`    | Use the Namecheap sandbox API instead of production      |

Examples:

```sh
# Second page, 50 per page, only domains matching "example"
cs-namecheap get-domains --page 2 --page-size 50 --search example

# Test against the sandbox environment
cs-namecheap get-domains --sandbox
```

### Output formats

These global flags work on any command:

| Flag            | Description                          |
| --------------- | ------------------------------------ |
| `--json`        | Output JSON                          |
| `--csv`         | Output CSV                           |
| `-v, --verbose` | Log to stderr                        |

```sh
cs-namecheap get-domains --json
cs-namecheap get-domains --csv > domains.csv
```

Dates (`Created`, `Expires`) are emitted in ISO 8601 (`yyyy-mm-dd`) format
across all output formats.

## Releasing

Releases are produced automatically by GitHub Actions
([`.github/workflows/release.yml`](.github/workflows/release.yml)) whenever a
semver tag is pushed. [GoReleaser](https://goreleaser.com) cross-compiles the
binaries and publishes them to a GitHub Release.

```sh
git tag v0.1.0
git push origin v0.1.0
```

## License

See [LICENSE](LICENSE).
