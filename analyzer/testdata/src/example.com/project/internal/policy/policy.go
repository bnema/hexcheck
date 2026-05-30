package policy

type Order struct {
	Total  int
	Status string
}

func (o *Order) CanApprove() bool { return o.Total > 0 }

type PolicyError struct{}

func (PolicyError) Error() string { return "policy" }
