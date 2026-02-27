package di

import (
	"context"
	"reflect"
)

// Service is the standard interface for managed services.
// Container auto-detects and calls these methods.
type Service interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// HealthChecker is the minimum interface for a DI-registered type to be
// recognized as a service. Types implementing this appear in the dashboard
// service list and participate in container health checks.
type HealthChecker interface {
	Health(ctx context.Context) error
}

// Namer is implemented by services that provide a custom display name.
// When not implemented, the container derives a name from the type
// using the pattern "{pkgPath}.{TypeName}".
type Namer interface {
	Name() string
}

// Starter is implemented by services that need initialization.
// The container calls Start during its Start phase or on first resolve.
type Starter interface {
	Start(ctx context.Context) error
}

// Stopper is implemented by services that need graceful shutdown.
// The container calls Stop during its Stop phase in reverse order.
type Stopper interface {
	Stop(ctx context.Context) error
}

// ServiceName returns the display name for a value.
// If the value implements Namer, its Name() is used.
// Otherwise, the name is derived from reflect.Type as "{pkgPath}.{TypeName}".
func ServiceName(v any) string {
	if n, ok := v.(Namer); ok {
		name := n.Name()
		if name != "" {
			return name
		}
	}
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.PkgPath() + "." + t.Name()
}

// Configurable is optional for services that need configuration.
type Configurable interface {
	Configure(config any) error
}

// Disposable is optional for scoped services that need cleanup.
type Disposable interface {
	Dispose() error
}
