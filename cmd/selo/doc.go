// Command selo is the CLI for the selo Brazilian-document toolkit. It derives
// one subcommand per registered document kind from the core registry, plus
// detect, person (synthetic identities), mcp (Model Context Protocol server),
// and version. Each kind subcommand supports --validate, --generate, --format,
// --origin (geolocatable kinds), --from FILE|- (bulk), --count, and --uf
// (UF-scoped kinds). It exits 1 when a document is invalid, so it is scriptable.
package main
