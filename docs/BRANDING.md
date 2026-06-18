# brdoc Branding Names

> Branding reference for the rebrand of the Brazilian-document toolkit (validate /
> generate / format / geolocate 11 document types + PIX, as a Go library + CLI + MCP
> server). Audience: Go developers building Brazilian fintech / gov / compliance software.
>
> **Rebrand-ready:** all brand strings live in `meta.go` (`AppName`, `CLIUse`,
> `MCPServerName`); applying a chosen name = update `meta.go` + go.mod module path +
> import rewrite + binary/GoReleaser names.

## Project Name Candidates

| # | Name | Package / CLI (ASCII) | Rationale |
|---|------|------------------------|-----------|
| 1 | **brdoc** (current) | `brdoc` | Descriptive ("BR doc(uments)"). Clear, but generic and collides conceptually with `paemuri/brdoc`. |
| 2 | **Cartório** ⭐ | `cartorio` | A *cartório* is the Brazilian notary office that issues and authenticates documents — the exact real-world analogue of this toolkit. Evocative, unmistakably BR, memorable. **Recommended.** |
| 3 | **Selo** | `selo` | "Seal/stamp" — the mark of validity. Short, clean, brandable; speaks to validation. |
| 4 | **Cédula** | `cedula` | "ID card / banknote" — core identity-document imagery. |
| 5 | **Crivo** | `crivo` | "Sieve / scrutiny" (*passar pelo crivo* = to scrutinize). Distinctive, captures validation/filtering. |
| 6 | **Confere** | `confere` | "Checks out / it's correct." Friendly verb that maps to validation results; great CLI feel (`confere cpf …`). |
| 7 | **Autentica** | `autentica` | "Authenticate." Compliance-flavored, descriptive. |
| 8 | **Registro** | `registro` | "Registry/record" — doubles as a nod to the internal registry architecture and *Registro Geral* (RG). |
| 9 | **Documenta** | `documenta` | Portmanteau of *documento*; descriptive and approachable. |
| 10 | **Carimbo** | `carimbo` | "Rubber stamp" — the act of validating/approving a document. Playful, concrete. |
| 11 | **Verdoc** | `verdoc` | Compound: *ver*(ify) + *doc*. Short, technical. |
| 12 | **Aferir** | `aferir` | "To gauge / verify against a standard." Precise, compliance-oriented. |

**Top pick:** **Cartório** (`cartorio`) — strongest concept fit (the place documents are made and verified), clean ASCII package/CLI/module name, no known Go-ecosystem collision. Runner-ups: **Selo**, **Crivo**, **Confere**.

## Feature Names

| Feature | Current Name | Branded Name Options |
|---------|-------------|----------------------|
| Validate a document | `Validate` | `Confere`, `Aferir`, `Check` |
| Generate fake-but-valid data | `Generate` | `Emitir` ("issue"), `Forjar` ("forge"), `Mint` |
| Format / mask | `Format` | `Carimbar` ("stamp"), `Mask`, `Formatar` |
| Auto-detect kind | `Detect` | `Reconhecer` ("recognize"), `Identify`, `Triagem` |
| Resolve issuing UF | `Origin` | `Procedência`, `Locate`, `UF` |
| Bulk file/stdin validation | `--from` | `Lote` ("batch"), `Stream`, `Bulk` |
| Fake-person generator (planned) | `GenPerson` | `Cidadão` ("citizen"), `Persona`, `Ficha` |

## Component Names

| Component | Branded Name Options |
|-----------|----------------------|
| `Document` interface + registry | `Cartório`, `Registro`, `Acervo` |
| MCP server adapter | `Balcão` ("service counter"), `Atendente`, `Bridge` |
| paemuri-compat layer | `Ponte` ("bridge"), `Compat`, `Legado` |
| UF lookup tables (CEP/DDD/region) | `Mapa`, `Atlas`, `Geo` |
| CLI | `Terminal`, `Guichê` ("ticket window"), `Console` |

## Taglines

- **Cartório for your code.**
- Validate, generate, and format every Brazilian document.
- One registry. Eleven documents. Three surfaces.
- The notary office, as a Go package.
- CPF to PIX — checked, minted, and masked.
- Brazilian documents, done right — library, CLI, and MCP.
- Stop trusting regex. Start trusting the check digits.
- Compliance-grade Brazilian document validation for Go.

## CLI Branding Themes

The per-kind subcommands stay document-named (`cpf`, `cnpj`, …); these themes reshape the
top-level verbs and help framing.

**Theme A — Cartório (notary):**
```
<brand> cpf --emitir          # generate  ("issue")
<brand> cpf --conferir VALUE  # validate  ("check")
<brand> cpf --carimbar VALUE  # format    ("stamp")
<brand> reconhecer VALUE      # detect    ("recognize")
<brand> balcao                # mcp       ("service counter" / server)
```

**Theme B — Minimal (verbs, English):**
```
<brand> cpf -g | -v VALUE | --format VALUE | --origin VALUE
<brand> detect VALUE
<brand> serve                 # mcp
```

**Theme C — Pipeline (data):**
```
<brand> cpf --mint            # generate
<brand> cpf --check VALUE     # validate
<brand> cpf --mask VALUE      # format
<brand> classify VALUE        # detect
```

> Recommendation: keep the current flag-based UX (Theme B) for stability; adopt the
> Cartório metaphor (Theme A) only in docs/marketing copy.

## Color Palette Suggestions

Brazil-inspired but restrained for a developer tool (flag green/blue/gold, desaturated).

| Role | Name | Hex |
|------|------|-----|
| Primary | Verde Cartório (deep green) | `#1B7A43` |
| Secondary | Azul Selo (flag navy) | `#15396B` |
| Accent | Ouro (gold) | `#E8B021` |
| Warning | Carmim (alert red) | `#C8442B` |
| Muted | Ardósia (slate) | `#64748B` |

## Logo Concepts

- **Carimbo (rubber stamp):** a circular notary-stamp outline enclosing a check mark — the toolkit "stamps" documents as valid.
- **Selo + dígito:** a wax-seal/badge shape whose center is a stylized check digit, nodding to mod-11 verification.
- **Documento dobrado:** a folded-document glyph with a green check fold, in the green→navy gradient.
- **Losango (flag diamond):** the Brazilian-flag central diamond abstracted into a monogram of the chosen initial (e.g., a "C" for Cartório).

### Brand icon command (IconForge)

If `iconforge` is unavailable, generate later with:
```
iconforge forge --generate --name cartorio --primary "#1B7A43" --secondary "#15396B" --accent "#E8B021" --output build/icons
```
