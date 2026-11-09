# asset-sync

Pushes an inventory CSV to the asset service in batches.

Requires Go 1.22 or newer. Run from the repository directory:

```sh
go run . --input inventory.csv --dry-run
go run . --input inventory.csv
```

The CSV columns are `id,name,owner,site,status`. Empty IDs and duplicate IDs are rejected. `--config` selects another configuration file; the default is `config.json`.
