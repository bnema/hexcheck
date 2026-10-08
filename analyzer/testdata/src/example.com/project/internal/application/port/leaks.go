package port

import (
	"example.com/framework"
	"example.com/project/internal/infrastructure/sql" // want "no-infra-imports-in-ports"
)

type Box[T any] struct{ Value T }

type Streams interface {
	Rows() <-chan sql.Row                     // want "no-infra-types-in-ports"
	Each(fn func(*sql.Row) error) error       // want "no-infra-types-in-ports"
	Boxed() Box[*framework.Context]           // want "no-infra-types-in-ports"
	Inline() struct{ Ctx *framework.Context } // want "no-infra-types-in-ports"
	Plain(id string) error
}

type Ignored interface {
	//hexcheck:ignore no-infra-types-in-ports transport row is part of the published contract
	Raw() sql.Row
}

type SameLine interface {
	Raw() sql.Row //hexcheck:ignore no-infra-types-in-ports published contract
}

type BadDirective interface {
	Raw() sql.Row /* want "no-infra-types-in-ports" "invalid-ignore-directive: hexcheck:ignore needs a rule list and a reason" */ //hexcheck:ignore no-infra-types-in-ports
}

type UnknownDirective interface {
	Raw() sql.Row /* want "no-infra-types-in-ports" "invalid-ignore-directive: hexcheck:ignore names unknown rule not-a-rule" */ //hexcheck:ignore not-a-rule some reason
}

type OtherRuleDirective interface {
	Raw() sql.Row /* want "no-infra-types-in-ports" */ //hexcheck:ignore no-adapter-imports-in-core wrong rule does not suppress
}
