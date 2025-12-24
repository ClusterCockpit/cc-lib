# Agent Guidelines for cc-lib

## Build, Test & Lint

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test -v ./ccMessage

# Run single test
go test -v ./ccMessage -run TestJSONEncode

# Run with coverage
go test -cover ./...

# Build (library only, no binary)
go build ./...
```

## Code Style

**Copyright Header:** All files must start with:
```go
// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
```

**Package Comments:** Every package requires a package-level comment describing its purpose (linter enforced).

**Imports:** Group stdlib first, then third-party with blank line separator. Use common aliases:
```go
import (
    "encoding/json"
    "time"

    cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
    lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
    mp "github.com/ClusterCockpit/cc-lib/v2/messageProcessor"
)
```

**Naming:** PascalCase for exported (public), camelCase for unexported (private). Avoid stuttering (e.g., `cache.New()` not `cache.NewCache()`).

**Error Handling:** Return errors, don't panic. Check all error returns. Use descriptive error messages with context.

**Comments:** Godoc-style for all exported functions, types, constants. Start with the item name: `// NewMessage creates...`

**Thread Safety:** Use `sync.Mutex` for shared state. Document thread-safety guarantees in comments.

**Types:** Prefer strongly-typed over `interface{}` when possible. Use `any` instead of `interface{}` for modern Go.

**Config Structs:** Use JSON tags, embed `default*Config` for common fields, support `omitempty` for optional fields.

**Testing:** Name tests `Test<FunctionName>_<Scenario>`. Use table-driven tests for multiple cases. Check both success and error paths.
