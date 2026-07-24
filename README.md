# collate

Rebuild duplex documents from simplex PDF scans.

## Usage

```sh
collate merge [flags] -f <front.pdf> -b <back.pdf> -o <output.pdf> 
```

`back.pdf` is assumed to be in reverse order by default, producing `front 1, back N, front 2, back N-1, ...`. Use `--backorder=forward` when back pages are in forward order:

```sh
collate merge -f front.pdf -b back.pdf -o output.pdf --backorder=forward
```

List scanners available to collate:

```sh
collate scanner list
```

Scan one page from a listed scanner into a PDF:

```sh
collate scan --device 'airscan:e0:OfficeJet' --output document.pdf
```

Force DIN A4 scan geometry and A4-sized PDF pages:

```sh
collate scan --device 'airscan:e0:OfficeJet' --paper a4 --output document.pdf
```

Scan all pages loaded in a feeder by naming its source and opting into batch mode:

```sh
collate scan --device 'airscan:e0:OfficeJet' --source ADF --batch --paper a4 --output document.pdf
```

The exact feeder source name is device-specific; inspect it with `scanimage --device-name '<device>' --help`. The only supported paper format is currently `a4`. On Linux, scanner discovery and scanning use SANE's `scanimage` command, commonly provided by the `sane-utils` package. Device visibility may also depend on SANE backend support, device permissions, udev rules, or network configuration.

## Install

Download the archive for your operating system and CPU architecture from [GitHub Releases](https://github.com/slobbe/collate/releases).

### Building from source

To build from source, clone the repository and run `go build` in the `collate` directory:

```sh
git clone https://github.com/slobbe/collate.git
cd collate
go build -o bin/collate ./cmd/collate
```

## Development

To release, tag a commit on `main` after its CI run succeeds:

```sh
git tag v0.1.0
git push origin v0.1.0
```

## License

[MIT License](LICENSE)
