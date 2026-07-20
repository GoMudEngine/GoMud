package relationships

type Type string

const (
	TypeFamily   Type = "family"
	TypeFriend   Type = "friend"
	TypeRival    Type = "rival"
	TypeLover    Type = "lover"
	TypeEmployer Type = "employer" // I am their employer
	TypeEmployee Type = "employee" // I am their employee (auto-mirror only)
)

// IsSymmetric returns true for relationship types where the reverse
// edge has the same type (family, friend, rival, lover) and false
// for asymmetric pairs (employer/employee).
func IsSymmetric(t Type) bool {
	switch t {
	case TypeFamily, TypeFriend, TypeRival, TypeLover:
		return true
	}
	return false
}

// InverseType returns the auto-mirror reverse type. Symmetric types
// return themselves; asymmetric pairs flip.
func InverseType(t Type) Type {
	switch t {
	case TypeEmployer:
		return TypeEmployee
	case TypeEmployee:
		return TypeEmployer
	}
	return t // symmetric: same type
}

// Relation is one outgoing edge from a mob to another mob.
type Relation struct {
	Other   int // mob template id of the other party
	Type    Type
	Subtype string // optional flavor; "" if unset
}
