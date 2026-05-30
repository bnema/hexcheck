package domain

import "example.com/framework"

type Service struct {
	Context *framework.Context // want "no-framework-types-in-core"
}
