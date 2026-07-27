package identity

import "context"

type CustomerContext struct {
	TenantID       string
	SessionID      string
	CustomerID     string
	Locale         string
	DataGeneration int
}

type customerContextKey struct{}

func WithCustomer(ctx context.Context, customer CustomerContext) context.Context {
	return context.WithValue(ctx, customerContextKey{}, customer)
}

func CustomerFromContext(ctx context.Context) (CustomerContext, bool) {
	customer, ok := ctx.Value(customerContextKey{}).(CustomerContext)
	return customer, ok
}
