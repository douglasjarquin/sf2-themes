# Design

Catalog ids and the embedded catalog. Moved out of the README.

See [docs/theme-guidelines.md](docs/theme-guidelines.md), [docs/roster.md](docs/roster.md), and [docs/previews/](docs/previews/).

Theme data lives in [`themes/`](themes/). Dark themes keep their original IDs, and light variants use the corresponding `<id>-light` ID.
Those catalog ids remain the short CLI names; generated adapter identities add the `sf2-` prefix.
The binary embeds that catalog at build time through `themes/embed.go`, so a copied binary needs no sibling files.
