# collate

Rebuild duplex documents from simplex PDF scans.

```text
collate [flags] <front.pdf> <back.pdf> <output.pdf>
```

All PDF paths are normalized to clean absolute paths. Relative paths, including
bare filenames such as `front.pdf`, are resolved from the current working
directory. `~` and `~/...` are expanded to the current user's home directory,
even when the shell did not expand them first. Each input and output path must
have a `.pdf` extension (case-insensitive).

By default, `collate` assumes `back.pdf` was scanned in reverse page order and
writes pages as `front 1, back N, front 2, back N-1, ...`.

Use `-back-order=forward` when `back.pdf` is already in matching page order:

```sh
collate -back-order=forward ~/scans/front.pdf ~/scans/back.pdf ./output.pdf
```

Supported `-back-order` values are:

- `reverse` (default)
- `forward`

## Releases

Download the archive for your operating system and CPU architecture from the
[GitHub Releases](https://github.com/slobbe/collate/releases) page. Releases
include Linux, macOS, and Windows builds for AMD64 and ARM64, plus a
`checksums.txt` file for verification.

Linux and macOS archives use `tar.gz`; Windows archives use `zip`. Creating and
pushing a version tag, such as `v0.1.0`, triggers the release workflow only
when the tagged commit is contained in `main`:

```sh
git switch main
git pull --ff-only
git tag v0.1.0
git push origin v0.1.0
```
