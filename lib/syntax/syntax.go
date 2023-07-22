package syntax

import "github.com/sboehler/knut/lib/syntax/scanner"

type Pos = scanner.Range

type Commodity Pos

type Account Pos

type AccountMacro Pos

type Decimal Pos
