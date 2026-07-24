# Architecture

`gh-exhibit` follows onion architecture. This document describes the layers, the
dependency rule between them, and the boundary conventions that keep the rule
meaningful rather than cosmetic.

## Layers

```text
internal/
├── presentation/   # CLI entrypoint (cobra commands, flag parsing)
├── application/    # Orchestrates domain operations; no business rules of its own
├── domain/         # Entities, value objects, repository interfaces — the core
├── infrastructure/ # Implements domain repository interfaces (GitHub API, filesystem)
└── registry/       # Wires concrete infrastructure types into abstract types
```

## Dependency direction

```text
presentation → application → domain ← infrastructure
```

- `domain` depends on nothing else in this project.
- `application` depends only on `domain`.
- `infrastructure` implements interfaces defined in `domain/repositories`
  (dependency inversion) — it does not export its own concrete types.
- `presentation` depends only on `application`.

No layer imports a concrete type from a layer it doesn't own; cross-layer calls
go through the abstract types defined in `domain`.

## Infrastructure types stay unexported

An `internal/infrastructure` implementation must not export its struct type.
Only the interface it satisfies (defined in `internal/domain/repositories`) and
a `New...` constructor returning that interface are exported. This keeps the
concrete type substitutable and stops callers from depending on infrastructure
details the interface doesn't promise.

Wrong — the concrete type is exported, so callers can reference fields/methods
the interface doesn't declare:

```go
type EvidenceRepository struct {
    client requester
    sleep  sleeper
}

func NewEvidenceRepository(opts api.ClientOptions) (*EvidenceRepository, error) {
    client, err := api.NewRESTClient(opts)
    if err != nil {
        return nil, fmt.Errorf("new REST client: %w", err)
    }
    return &EvidenceRepository{client: client, sleep: realSleep}, nil
}
```

Right — the struct is unexported; the constructor returns the domain-layer
abstract type:

```go
type evidenceRepository struct {
    client requester
    sleep  sleeper
}

// NewEvidenceRepository returns the repositories.EvidenceRepository
// abstraction, hiding the concrete implementation.
func NewEvidenceRepository(opts api.ClientOptions) (repositories.EvidenceRepository, error) {
    client, err := api.NewRESTClient(opts)
    if err != nil {
        return nil, fmt.Errorf("new REST client: %w", err)
    }
    return &evidenceRepository{client: client, sleep: realSleep}, nil
}
```

See `internal/infrastructure/github` and `internal/infrastructure/persistence`
for existing implementations that follow this convention.

## Boundaries convert types, not just direction

Aligning dependency direction (inversion, coding to interfaces) is necessary
but not sufficient for change isolation. An interface shaped by the
infrastructure side's concerns (a GitHub API response shape, a filesystem
layout) still couples callers to that shape, regardless of which way the
arrow points. What actually stops a change from propagating past a layer is a
type conversion at the boundary — untrusted input (HTTP responses, file
contents) is decoded into a domain type before it travels further inward,
rather than merely validated and passed through unchanged. Value object
constructors (e.g. in `internal/domain/valueobjects`) are where this
conversion happens: a raw string/int can't cross into domain/application code
without going through one first.
