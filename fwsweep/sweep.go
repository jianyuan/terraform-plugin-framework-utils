// Package fwsweep provides helpers for writing acceptance test sweepers for
// Terraform Plugin Framework resources.
//
// Call Register from an init function or package-level var in a _test.go file,
// and wire up the sweepers in that package's TestMain:
//
//	func TestMain(m *testing.M) {
//		resource.TestMain(m)
//	}
//
// Then run them with "go test ./... -sweep=all". Use NewResource to delete
// items through the resource's own Delete method, or implement Sweepable when
// a direct API call is simpler.
package fwsweep

import (
	"context"
	"errors"

	"golang.org/x/sync/errgroup"
)

// DefaultConcurrency is the default maximum number of concurrent deletes.
const DefaultConcurrency = 10

// Sweepable is a single item that can be deleted by a sweeper.
type Sweepable interface {
	Delete(ctx context.Context) error
}

// SweeperFn collects the items to sweep using the given provider data.
type SweeperFn[P any] func(ctx context.Context, providerData P) ([]Sweepable, error)

type options struct {
	concurrency  int
	dependencies []string
}

// Option configures Sweep and Register.
type Option func(*options)

// WithConcurrency sets the maximum number of concurrent deletes. Values below 1
// mean no limit.
func WithConcurrency(n int) Option {
	return func(o *options) {
		o.concurrency = n
	}
}

// WithDependencies sets the names of sweepers that must run before this one.
// It only applies to Register.
func WithDependencies(dependencies ...string) Option {
	return func(o *options) {
		o.dependencies = append(o.dependencies, dependencies...)
	}
}

func newOptions(opts []Option) options {
	o := options{concurrency: DefaultConcurrency}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// Sweep deletes all sweepables concurrently and returns the joined errors of
// any that failed. A failure does not stop the remaining deletes.
func Sweep(ctx context.Context, sweepables []Sweepable, opts ...Option) error {
	o := newOptions(opts)
	errs := make([]error, len(sweepables))

	var g errgroup.Group
	if o.concurrency > 0 {
		g.SetLimit(o.concurrency)
	}
	for i, sweepable := range sweepables {
		g.Go(func() error {
			errs[i] = sweepable.Delete(ctx)
			return nil
		})
	}
	_ = g.Wait()

	return errors.Join(errs...)
}
