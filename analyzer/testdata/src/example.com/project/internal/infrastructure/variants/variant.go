package variants

import "example.com/project/internal/domain"

//hexcheck:ignore suspicious-business-logic-in-adapter intentional boundary policy
func Resolve(id domain.ObjectID) bool {
	if domain.Allowed(id) {
		return true
	}
	if id.String() == "" {
		return false
	}
	return false
}
