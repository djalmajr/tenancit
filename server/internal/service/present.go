package service

import (
	"github.com/centralit/resource-tenant/server/internal/crypto"
	"github.com/centralit/resource-tenant/server/internal/store/db"
)

// PresentValue is the exported wrapper over presentValue for admin handlers:
// masks secret values unless reveal is true (RN-06).
func PresentValue(c *crypto.Cryptor, isSecret bool, v db.TenantResourceValue, reveal bool) (string, error) {
	return presentValue(c, isSecret, v, reveal)
}
