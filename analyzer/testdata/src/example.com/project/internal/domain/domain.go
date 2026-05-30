package domain

import _ "example.com/project/internal/sqlbad" // want "no-adapter-imports-in-core"

type User struct{ Name string }
