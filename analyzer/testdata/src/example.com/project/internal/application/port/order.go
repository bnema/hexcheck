package port

type Inventory interface {
	Reserve() error
}

type Payments interface {
	Charge() error
}
