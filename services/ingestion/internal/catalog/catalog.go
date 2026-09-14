package catalog

import "github.com/google/uuid"

type Service struct {
	ID          uuid.UUID
	Slug        string
	Environment string
}

// Known local catalog. Matches Phase 1 seeds plus orders-api for the simulator.
// Ingestion does not query PostgreSQL on the hot path.
var known = []Service{
	{ID: uuid.MustParse("22222222-2222-4222-8222-222222222221"), Slug: "payments-api", Environment: "production"},
	{ID: uuid.MustParse("22222222-2222-4222-8222-222222222222"), Slug: "checkout-web", Environment: "production"},
	{ID: uuid.MustParse("22222222-2222-4222-8222-222222222223"), Slug: "inventory-worker", Environment: "production"},
	{ID: uuid.MustParse("22222222-2222-4222-8222-222222222224"), Slug: "auth-service", Environment: "production"},
	{ID: uuid.MustParse("22222222-2222-4222-8222-222222222225"), Slug: "notifications-worker", Environment: "production"},
	{ID: uuid.MustParse("22222222-2222-4222-8222-222222222226"), Slug: "orders-api", Environment: "production"},
}

func Resolve(id *uuid.UUID, slug, env string) (Service, bool) {
	if env == "" {
		env = "production"
	}
	if id != nil {
		for _, s := range known {
			if s.ID == *id {
				if slug != "" && slug != s.Slug {
					return Service{}, false
				}
				if env != s.Environment {
					return Service{}, false
				}
				return s, true
			}
		}
		return Service{}, false
	}
	if slug == "" {
		return Service{}, false
	}
	for _, s := range known {
		if s.Slug == slug && s.Environment == env {
			return s, true
		}
	}
	return Service{}, false
}
