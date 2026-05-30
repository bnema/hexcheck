package domain

type PolicyError struct{}

func (PolicyError) Error() string { return "policy" }
