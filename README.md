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
