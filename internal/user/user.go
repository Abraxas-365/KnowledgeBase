package user

type SubscriptionType string

const (
	FreeTier  SubscriptionType = "free-tier"
	LinkoPlus SubscriptionType = "linko-plus"
	LinkoVIP  SubscriptionType = "linko-vip"
)

type Role string

const (
	RoleAdmin     Role = "admin"
	RoleLinkoUser Role = "linko-user"
)

type User struct {
	ID               string           `json:"id" db:"id"`
	Email            string           `json:"email" db:"email"`
	IsAdmin          bool             `json:"isAdmin" db:"is_admin"`
	ProviderID       string           `json:"-" db:"provider_id"`
	Provider         string           `json:"-" db:"provider"`
	Role             Role             `json:"role" db:"role"`
	SubscriptionType SubscriptionType `json:"subscriptionType,omitempty" db:"subscription_type"`
}

func (au *User) GetID() string {
	return au.ID
}

// IsInRole checks if the user has the specified role
func (u *User) IsInRole(role Role) bool {
	return u.Role == role
}

// HasAdminAccess checks if the user has admin access
func (u *User) HasAdminAccess() bool {
	return u.IsAdmin || u.Role == RoleAdmin
}

// HasMinimumSubscription checks if user has at least the specified subscription level
func (u *User) HasMinimumSubscription(minLevel SubscriptionType) bool {
	if u.Role != RoleLinkoUser {
		return false
	}

	switch minLevel {
	case FreeTier:
		return true // All linko users have at least free tier
	case LinkoPlus:
		return u.SubscriptionType == LinkoPlus || u.SubscriptionType == LinkoVIP
	case LinkoVIP:
		return u.SubscriptionType == LinkoVIP
	default:
		return false
	}
}
