package seed

import (
	"context"
	"fmt"
	"strings"

	"github.com/Everything-is-Code/3scaleextract/internal/admin"
)

type Options struct {
	SkipExisting bool
	DryRun       bool
}

type Result struct {
	Backends     map[string]int
	Services     map[string]int
	Plans        map[string]int
	Applications []string
	AccountID    int
	Skipped      []string
}

type Seeder struct {
	client  admin.Client
	opts    Options
	skipped []string
}

func NewSeeder(client admin.Client, opts Options) *Seeder {
	return &Seeder{client: client, opts: opts}
}

func (s *Seeder) Run(ctx context.Context) (*Result, error) {
	backends, account, products := DefaultFixtures()
	return s.RunFixtures(ctx, backends, account, products)
}

// RunFixtures seeds the provided fixture set (built-in or loaded from YAML).
func (s *Seeder) RunFixtures(ctx context.Context, backends []BackendFixture, account AccountFixture, products []ProductFixture) (*Result, error) {
	result := &Result{
		Backends: make(map[string]int),
		Services: make(map[string]int),
		Plans:    make(map[string]int),
	}

	for _, b := range backends {
		id, err := s.ensureBackend(ctx, b)
		if err != nil {
			return result, fmt.Errorf("backend %q: %w", b.SystemName, err)
		}
		result.Backends[b.SystemName] = id
	}

	accountID, err := s.ensureAccount(ctx, account)
	if err != nil {
		return result, fmt.Errorf("account: %w", err)
	}
	result.AccountID = accountID

	for _, p := range products {
		if err := s.seedProduct(ctx, p, result, accountID); err != nil {
			return result, fmt.Errorf("product %q: %w", p.SystemName, err)
		}
	}
	result.Skipped = append(result.Skipped, s.skipped...)

	return result, nil
}

func (s *Seeder) resultSkip(label string) {
	s.skipped = append(s.skipped, label)
}

func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already been taken") ||
		strings.Contains(msg, "422") ||
		strings.Contains(msg, "has already been taken")
}
