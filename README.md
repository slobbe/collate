# collate

Create duplex PDFs with a simplex scanner.

## Usage

### Scanning

Collate uses the eSCL protocol to discover scanners on the local network.

Make an interactive scan:

```sh
collate scan [flags]
```

Optional flags:

- `--device <device>`: Select a scanner by its advertised name or device URL.
- `--source <source>`: Select an input source by ID or name, such as `Platen`, `ADF Simplex`, or `adf`.
- `--paper <paper>`: Set the paper format to `a4`, `a5`, or `letter`.
- `--mode <mode>`: Select an advertised color mode, such as `BlackAndWhite1`, `Grayscale8`, or `RGB24`.
- `--resolution <dpi>`: Set an advertised source-specific resolution in DPI, such as `300`.
- `--output <path>`: Set the output PDF path. The `.pdf` extension is added when omitted.

For example, to preselect an DIN-A4, ADF scan with 300 DPI resolution and save to `document.pdf`:

```sh
collate scan --source adf --paper a4 --resolution 300 --output document.pdf
```

List available scanners without scanning:

```sh
collate scan --device-list
```

### Merge existing PDFs

```sh
collate merge [flags] -f <front.pdf> -b <back.pdf> -o <output.pdf>
```

`back.pdf` is assumed to be in reverse order by default, producing `front 1, back N, front 2, back N-1, ...`. Use `--backorder=forward` when back pages are in forward order:

```sh
collate merge -f front.pdf -b back.pdf -o output.pdf --backorder=forward
```

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
