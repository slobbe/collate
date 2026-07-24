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

On Linux, scanner discovery uses SANE's `scanimage` command, commonly provided by the `sane-utils` package. Device visibility may also depend on SANE backend support, device permissions, udev rules, or network configuration.

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
