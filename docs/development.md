# Development

[日本語版](development-ja.md)

Development tasks use the [just](https://github.com/casey/just) task runner. Running `just` with no arguments lists the recipes.

```sh
just test      # go test ./...
just make      # builds native CLI binaries in bin/
just fonts     # downloads the IPAex fonts into fonts/ (does nothing if already downloaded)
just samples   # generates the sample PDFs into out/
```
