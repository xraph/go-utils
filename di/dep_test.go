package di

import (
	"reflect"
	"testing"
)

func TestDepMode_String(t *testing.T) {
	tests := []struct {
		name string
		mode DepMode
		want string
	}{
		{"eager", DepEager, "eager"},
		{"lazy", DepLazy, "lazy"},
		{"optional", DepOptional, "optional"},
		{"lazy_optional", DepLazyOptional, "lazy_optional"},
		{"unknown", DepMode(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("DepMode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDepMode_IsLazy(t *testing.T) {
	tests := []struct {
		name string
		mode DepMode
		want bool
	}{
		{"eager_not_lazy", DepEager, false},
		{"lazy_is_lazy", DepLazy, true},
		{"optional_not_lazy", DepOptional, false},
		{"lazy_optional_is_lazy", DepLazyOptional, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.IsLazy(); got != tt.want {
				t.Errorf("DepMode.IsLazy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDepMode_IsOptional(t *testing.T) {
	tests := []struct {
		name string
		mode DepMode
		want bool
	}{
		{"eager_not_optional", DepEager, false},
		{"lazy_not_optional", DepLazy, false},
		{"optional_is_optional", DepOptional, true},
		{"lazy_optional_is_optional", DepLazyOptional, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.IsOptional(); got != tt.want {
				t.Errorf("DepMode.IsOptional() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEager(t *testing.T) {
	dep := Eager("test-service")

	if dep.Name != "test-service" {
		t.Errorf("Eager().Name = %v, want %v", dep.Name, "test-service")
	}

	if dep.Mode != DepEager {
		t.Errorf("Eager().Mode = %v, want %v", dep.Mode, DepEager)
	}

	if dep.Type != nil {
		t.Errorf("Eager().Type = %v, want nil", dep.Type)
	}
}

func TestEagerTyped(t *testing.T) {
	type TestService struct{}

	dep := EagerTyped[*TestService]("test-service")

	if dep.Name != "test-service" {
		t.Errorf("EagerTyped().Name = %v, want %v", dep.Name, "test-service")
	}

	if dep.Mode != DepEager {
		t.Errorf("EagerTyped().Mode = %v, want %v", dep.Mode, DepEager)
	}

	expectedType := reflect.TypeFor[*TestService]()
	if dep.Type != expectedType {
		t.Errorf("EagerTyped().Type = %v, want %v", dep.Type, expectedType)
	}
}

func TestLazy(t *testing.T) {
	dep := Lazy("test-service")

	if dep.Name != "test-service" {
		t.Errorf("Lazy().Name = %v, want %v", dep.Name, "test-service")
	}

	if dep.Mode != DepLazy {
		t.Errorf("Lazy().Mode = %v, want %v", dep.Mode, DepLazy)
	}

	if dep.Type != nil {
		t.Errorf("Lazy().Type = %v, want nil", dep.Type)
	}
}

func TestLazyTyped(t *testing.T) {
	type TestService struct{}

	dep := LazyTyped[*TestService]("test-service")

	if dep.Name != "test-service" {
		t.Errorf("LazyTyped().Name = %v, want %v", dep.Name, "test-service")
	}

	if dep.Mode != DepLazy {
		t.Errorf("LazyTyped().Mode = %v, want %v", dep.Mode, DepLazy)
	}

	expectedType := reflect.TypeFor[*TestService]()
	if dep.Type != expectedType {
		t.Errorf("LazyTyped().Type = %v, want %v", dep.Type, expectedType)
	}
}

func TestOptional(t *testing.T) {
	dep := Optional("test-service")

	if dep.Name != "test-service" {
		t.Errorf("Optional().Name = %v, want %v", dep.Name, "test-service")
	}

	if dep.Mode != DepOptional {
		t.Errorf("Optional().Mode = %v, want %v", dep.Mode, DepOptional)
	}

	if dep.Type != nil {
		t.Errorf("Optional().Type = %v, want nil", dep.Type)
	}
}

func TestOptionalTyped(t *testing.T) {
	type TestService struct{}

	dep := OptionalTyped[*TestService]("test-service")

	if dep.Name != "test-service" {
		t.Errorf("OptionalTyped().Name = %v, want %v", dep.Name, "test-service")
	}

	if dep.Mode != DepOptional {
		t.Errorf("OptionalTyped().Mode = %v, want %v", dep.Mode, DepOptional)
	}

	expectedType := reflect.TypeFor[*TestService]()
	if dep.Type != expectedType {
		t.Errorf("OptionalTyped().Type = %v, want %v", dep.Type, expectedType)
	}
}

func TestLazyOptional(t *testing.T) {
	dep := LazyOptional("test-service")

	if dep.Name != "test-service" {
		t.Errorf("LazyOptional().Name = %v, want %v", dep.Name, "test-service")
	}

	if dep.Mode != DepLazyOptional {
		t.Errorf("LazyOptional().Mode = %v, want %v", dep.Mode, DepLazyOptional)
	}

	if dep.Type != nil {
		t.Errorf("LazyOptional().Type = %v, want nil", dep.Type)
	}
}

func TestLazyOptionalTyped(t *testing.T) {
	type TestService struct{}

	dep := LazyOptionalTyped[*TestService]("test-service")

	if dep.Name != "test-service" {
		t.Errorf("LazyOptionalTyped().Name = %v, want %v", dep.Name, "test-service")
	}

	if dep.Mode != DepLazyOptional {
		t.Errorf("LazyOptionalTyped().Mode = %v, want %v", dep.Mode, DepLazyOptional)
	}

	expectedType := reflect.TypeFor[*TestService]()
	if dep.Type != expectedType {
		t.Errorf("LazyOptionalTyped().Type = %v, want %v", dep.Type, expectedType)
	}
}

func TestDepNames(t *testing.T) {
	tests := []struct {
		name string
		deps []Dep
		want []string
	}{
		{
			name: "empty",
			deps: []Dep{},
			want: []string{},
		},
		{
			name: "single",
			deps: []Dep{Eager("service1")},
			want: []string{"service1"},
		},
		{
			name: "multiple",
			deps: []Dep{
				Eager("service1"),
				Lazy("service2"),
				Optional("service3"),
			},
			want: []string{"service1", "service2", "service3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DepNames(tt.deps)

			if len(got) != len(tt.want) {
				t.Errorf("DepNames() length = %v, want %v", len(got), len(tt.want))

				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("DepNames()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestDepsFromNames(t *testing.T) {
	tests := []struct {
		name  string
		names []string
		want  []Dep
	}{
		{
			name:  "empty",
			names: []string{},
			want:  []Dep{},
		},
		{
			name:  "single",
			names: []string{"service1"},
			want:  []Dep{Eager("service1")},
		},
		{
			name:  "multiple",
			names: []string{"service1", "service2", "service3"},
			want: []Dep{
				Eager("service1"),
				Eager("service2"),
				Eager("service3"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DepsFromNames(tt.names)

			if len(got) != len(tt.want) {
				t.Errorf("DepsFromNames() length = %v, want %v", len(got), len(tt.want))

				return
			}

			for i := range got {
				if got[i].Name != tt.want[i].Name {
					t.Errorf("DepsFromNames()[%d].Name = %v, want %v", i, got[i].Name, tt.want[i].Name)
				}

				if got[i].Mode != tt.want[i].Mode {
					t.Errorf("DepsFromNames()[%d].Mode = %v, want %v", i, got[i].Mode, tt.want[i].Mode)
				}
			}
		})
	}
}

func TestDepRoundTrip(t *testing.T) {
	// Test that converting names -> deps -> names is lossless
	original := []string{"service1", "service2", "service3"}
	deps := DepsFromNames(original)
	result := DepNames(deps)

	if len(result) != len(original) {
		t.Fatalf("RoundTrip length = %v, want %v", len(result), len(original))
	}

	for i := range original {
		if result[i] != original[i] {
			t.Errorf("RoundTrip[%d] = %v, want %v", i, result[i], original[i])
		}
	}
}
