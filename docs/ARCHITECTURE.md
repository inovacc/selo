# Architecture

`github.com/inovacc/selo` is a Go toolkit for Brazilian documents exposed through three surfaces
(library, CLI, MCP server) over one core: a `Document` interface plus a self-registering type
registry. The CLI and MCP server derive their entire surface from the registry, so adding a
document type requires no changes to them.

## System overview

```mermaid
flowchart TB
    subgraph surfaces["Surfaces"]
        CLI["CLI — cmd/selo<br/>(Cobra; subcommand per Kind)"]
        MCP["MCP server — mcp<br/>(stdio; 6 tools)"]
        LIB["Library API<br/>(NewCPF()… + registry funcs)"]
        COMPAT["compat<br/>(paemuri/brdoc Is* drop-in)"]
    end

    subgraph core["Core package: selo"]
        REG["Registry<br/>Register / Get / Kinds<br/>Validate / Generate / Format / Detect"]
        IFACE["Document interface<br/>+ OriginResolver (opt)<br/>+ UFScoped (opt)"]
        TYPES["Document types<br/>CPF, CNPJ, CNH, PIS, RENAVAM,<br/>VoterID, CEP, Phone, Plate, CNS,<br/>RG, IE, PIX"]
        PERSON["GeneratePerson<br/>(coherent synthetic identity)"]
    end

    CLI --> REG
    MCP --> REG
    LIB --> REG
    LIB --> TYPES
    COMPAT --> TYPES
    PERSON --> TYPES
    REG -->|dispatch by Kind| TYPES
    TYPES -.implement.-> IFACE
    REG -.consumes.-> IFACE

    AGENT["AI agent"] -->|JSON-RPC / stdio| MCP
    USER["User / script"] -->|argv, exit codes| CLI
```

## Type registration lifecycle

Each document type registers itself at package-init time; the registry is fully populated before
`main` runs, so the CLI and MCP build their surfaces from `Kinds()`.

```mermaid
sequenceDiagram
    participant Go as Go runtime
    participant Type as type init() (cpf.go, ie.go, …)
    participant Reg as registry (selo)
    participant CLI as cmd/selo
    participant MCP as mcp server

    Go->>Type: import selo → run all init()
    loop one per document type
        Type->>Reg: Register(&CPF{}) / Register(&IE{}) …
        Reg->>Reg: store by Kind()
    end
    Note over Reg: registry now complete

    alt CLI start
        CLI->>Reg: Kinds()
        Reg-->>CLI: sorted kinds
        CLI->>CLI: build one subcommand per kind<br/>(+ --uf flag for UFScoped kinds)
    else MCP start
        MCP->>Reg: Kinds()
        Reg-->>MCP: sorted kinds (tool enums)
        MCP->>MCP: register 6 tools
    end
```

## Validate request flow (CLI)

```mermaid
sequenceDiagram
    participant U as User/script
    participant CLI as cmd/selo
    participant Reg as registry
    participant Doc as Document (e.g. CPF/RG)

    U->>CLI: selo cpf --validate 529.982.247-25
    CLI->>Reg: Validate(KindCPF, value)
    Reg->>Reg: Get(KindCPF)
    Reg->>Doc: Validate(value)
    Doc->>Doc: clean + check digits
    Doc-->>Reg: bool
    Reg-->>CLI: (bool, err)
    alt valid
        CLI-->>U: "valid" (exit 0)
    else invalid or error
        CLI-->>U: "invalid" (exit 1)
    end

    Note over CLI,Doc: UF-scoped kinds (RG, IE) with --uf call ValidateUF(value, uf);<br/>unsupported UF → ErrUFNotImplemented
```

## MCP tool-call flow

```mermaid
sequenceDiagram
    participant Agent as AI agent
    participant MCP as mcp server (stdio)
    participant Reg as registry
    participant Doc as Document

    Agent->>MCP: call validate_document {kind, value}
    MCP->>Reg: Validate(kind, value)
    Reg->>Doc: Validate(value)
    Doc-->>Reg: bool
    Reg-->>MCP: (bool, err)
    alt err (e.g. unknown kind)
        MCP-->>Agent: result.IsError = true ("selo mcp: …")
    else ok
        MCP-->>Agent: TextContent {valid: bool}
    end
    Note over MCP: logs → stderr; JSON-RPC → stdin/stdout
```

## Synthetic identity (GeneratePerson)

```mermaid
flowchart LR
    OPT["Options<br/>WithUF / WithVehicle / WithCompany"] --> GEN["GeneratePerson"]
    GEN --> UF{"UF chosen<br/>(explicit or random)"}
    UF --> CPF["CPF (region matches UF)"]
    UF --> VOTER["Voter ID (UF code)"]
    UF --> PHONE["Phone (DDD in UF)"]
    UF --> CEP["CEP (range in UF)"]
    GEN --> REST["RG, CNH, PIS, RENAVAM, CNS, PIX keys<br/>(+ Vehicle, Company if requested)"]
    CPF & VOTER & PHONE & CEP & REST --> P["Person (all valid, UF-consistent)"]
```

## Packages

| Package | Path | Responsibility |
|---------|------|----------------|
| core | `github.com/inovacc/selo` | `Document` interface, registry, all document types, `GeneratePerson`, errors |
| CLI | `cmd/selo` | Cobra CLI; registry-derived subcommands; `detect`, `person`, `mcp`, `version` |
| MCP | `mcp` | stdio Model Context Protocol server; 6 registry-backed tools |
| compat | `compat` | drop-in mirror of `paemuri/brdoc` v3 `Is*` API + signature-parity guard |

## Key design decisions
See the ADRs: [interface + registry architecture](adr/0001-interface-registry-architecture.md) and
[paemuri compat layer](adr/0002-paemuri-compat-layer.md).
