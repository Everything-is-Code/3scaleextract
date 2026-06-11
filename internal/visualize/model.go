package visualize

import (
	"github.com/Everything-is-Code/3scaleextract/internal/output"
)

// Tenant is the normalized in-memory view of an export directory.
type Tenant struct {
	Manifest     output.Manifest
	Products     []Product
	Backends     []Backend
	Applications []Application

	backendByID     map[int]*Backend
	productBySvcID  map[int]*Product
	planByID        map[int]*ApplicationPlan
}

// Product holds per-product export data with resolved backend links.
type Product struct {
	SystemName       string
	DisplayName      string
	ServiceID        int
	AuthType         string
	Policies         []Policy
	BackendUsages    []BackendUsage
	Plans            []ApplicationPlan
	OIDC             *OIDCConfiguration
	MissingFiles     []string
	StagingEndpoint  string
	ProductionEndpoint string
}

// Backend is a tenant backend API entry.
type Backend struct {
	ID              int
	SystemName      string
	Name            string
	PrivateEndpoint string
	ReferencedBy    []string
}

// BackendUsage links a product to a backend with a path prefix.
type BackendUsage struct {
	BackendID int
	Path      string
	Backend   *Backend
}

// Policy is one entry in a product policy chain.
type Policy struct {
	Name string
}

// ApplicationPlan is a product application plan.
type ApplicationPlan struct {
	ID         int
	SystemName string
	Name       string
	State      string
}

// OIDCConfiguration holds OIDC proxy settings when present.
type OIDCConfiguration struct {
	IssuerType     string
	IssuerEndpoint string
	Fields         map[string]string
}

// Application is a tenant application with resolved product name.
type Application struct {
	ID          int
	Name        string
	ServiceID   int
	PlanID      int
	AccountID   int
	State       string
	ProductName string
	PlanName    string
}

func (t *Tenant) BackendByID(id int) *Backend {
	if t == nil {
		return nil
	}
	return t.backendByID[id]
}

func (t *Tenant) ProductByServiceID(id int) *Product {
	if t == nil {
		return nil
	}
	return t.productBySvcID[id]
}

func (t *Tenant) PlanByID(id int) *ApplicationPlan {
	if t == nil {
		return nil
	}
	return t.planByID[id]
}

func (t *Tenant) index() {
	t.backendByID = make(map[int]*Backend, len(t.Backends))
	for i := range t.Backends {
		t.backendByID[t.Backends[i].ID] = &t.Backends[i]
	}

	t.productBySvcID = make(map[int]*Product, len(t.Products))
	t.planByID = make(map[int]*ApplicationPlan)
	for i := range t.Products {
		p := &t.Products[i]
		if p.ServiceID > 0 {
			t.productBySvcID[p.ServiceID] = p
		}
		for j := range p.Plans {
			plan := &p.Plans[j]
			if plan.ID > 0 {
				t.planByID[plan.ID] = plan
			}
		}
	}
}

func (t *Tenant) resolveReferences() {
	for i := range t.Products {
		p := &t.Products[i]
		for j := range p.BackendUsages {
			usage := &p.BackendUsages[j]
			usage.Backend = t.BackendByID(usage.BackendID)
			if usage.Backend != nil {
				usage.Backend.ReferencedBy = appendUnique(usage.Backend.ReferencedBy, p.SystemName)
			}
		}
	}

	for i := range t.Applications {
		app := &t.Applications[i]
		if prod := t.ProductByServiceID(app.ServiceID); prod != nil {
			app.ProductName = prod.SystemName
		}
		if plan := t.PlanByID(app.PlanID); plan != nil {
			app.PlanName = plan.Name
		}
	}
}

func appendUnique(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}
