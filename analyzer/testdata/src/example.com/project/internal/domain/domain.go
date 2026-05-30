package domain

import _ "example.com/project/internal/infrastructure/http" // want "no-adapter-imports-in-core"

type User struct{ Name string }
