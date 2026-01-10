package di

import (
	"testing"
)

func TestSingleton(t *testing.T) {
	opt := Singleton()

	if opt.Lifecycle != "singleton" {
		t.Errorf("Singleton().Lifecycle = %v, want %v", opt.Lifecycle, "singleton")
	}
}

func TestTransient(t *testing.T) {
	opt := Transient()

	if opt.Lifecycle != "transient" {
		t.Errorf("Transient().Lifecycle = %v, want %v", opt.Lifecycle, "transient")
	}
}

func TestScoped(t *testing.T) {
	opt := Scoped()

	if opt.Lifecycle != "scoped" {
		t.Errorf("Scoped().Lifecycle = %v, want %v", opt.Lifecycle, "scoped")
	}
}

func TestWithDependencies(t *testing.T) {
	opt := WithDependencies("service1", "service2", "service3")

	if len(opt.Dependencies) != 3 {
		t.Errorf("WithDependencies() length = %v, want 3", len(opt.Dependencies))
	}

	expected := []string{"service1", "service2", "service3"}
	for i, dep := range opt.Dependencies {
		if dep != expected[i] {
			t.Errorf("WithDependencies()[%d] = %v, want %v", i, dep, expected[i])
		}
	}
}

func TestWithDeps(t *testing.T) {
	deps := []Dep{
		Eager("service1"),
		Lazy("service2"),
		Optional("service3"),
	}
	opt := WithDeps(deps...)

	if len(opt.Deps) != 3 {
		t.Errorf("WithDeps() length = %v, want 3", len(opt.Deps))
	}

	for i, dep := range opt.Deps {
		if dep.Name != deps[i].Name {
			t.Errorf("WithDeps()[%d].Name = %v, want %v", i, dep.Name, deps[i].Name)
		}

		if dep.Mode != deps[i].Mode {
			t.Errorf("WithDeps()[%d].Mode = %v, want %v", i, dep.Mode, deps[i].Mode)
		}
	}
}

func TestWithDIMetadata(t *testing.T) {
	opt := WithDIMetadata("key1", "value1")

	if len(opt.Metadata) != 1 {
		t.Errorf("WithDIMetadata() length = %v, want 1", len(opt.Metadata))
	}

	if opt.Metadata["key1"] != "value1" {
		t.Errorf("WithDIMetadata()[\"key1\"] = %v, want %v", opt.Metadata["key1"], "value1")
	}
}

func TestWithGroup(t *testing.T) {
	opt := WithGroup("handlers")

	if len(opt.Groups) != 1 {
		t.Errorf("WithGroup() length = %v, want 1", len(opt.Groups))
	}

	if opt.Groups[0] != "handlers" {
		t.Errorf("WithGroup()[0] = %v, want %v", opt.Groups[0], "handlers")
	}
}

func TestMergeOptions_Empty(t *testing.T) {
	result := MergeOptions([]RegisterOption{})

	if result.Lifecycle != "singleton" {
		t.Errorf("MergeOptions([]).Lifecycle = %v, want singleton", result.Lifecycle)
	}

	if result.Metadata == nil {
		t.Error("MergeOptions([]).Metadata should not be nil")
	}
}

func TestMergeOptions_Single(t *testing.T) {
	opts := []RegisterOption{
		Transient(),
	}

	result := MergeOptions(opts)

	if result.Lifecycle != "transient" {
		t.Errorf("MergeOptions().Lifecycle = %v, want transient", result.Lifecycle)
	}
}

func TestMergeOptions_Multiple(t *testing.T) {
	opts := []RegisterOption{
		Singleton(),
		WithDependencies("service1", "service2"),
		WithDeps(Lazy("service3")),
		WithDIMetadata("key1", "value1"),
		WithDIMetadata("key2", "value2"),
		WithGroup("group1"),
		WithGroup("group2"),
	}

	result := MergeOptions(opts)

	if result.Lifecycle != "singleton" {
		t.Errorf("MergeOptions().Lifecycle = %v, want singleton", result.Lifecycle)
	}

	if len(result.Dependencies) != 2 {
		t.Errorf("MergeOptions().Dependencies length = %v, want 2", len(result.Dependencies))
	}

	if len(result.Deps) != 1 {
		t.Errorf("MergeOptions().Deps length = %v, want 1", len(result.Deps))
	}

	if len(result.Metadata) != 2 {
		t.Errorf("MergeOptions().Metadata length = %v, want 2", len(result.Metadata))
	}

	if len(result.Groups) != 2 {
		t.Errorf("MergeOptions().Groups length = %v, want 2", len(result.Groups))
	}
}

func TestMergeOptions_LifecycleOverride(t *testing.T) {
	opts := []RegisterOption{
		Singleton(),
		Transient(),
		Scoped(),
	}

	result := MergeOptions(opts)

	// Last one wins
	if result.Lifecycle != "scoped" {
		t.Errorf("MergeOptions().Lifecycle = %v, want scoped", result.Lifecycle)
	}
}

func TestMergeOptions_MetadataMerge(t *testing.T) {
	opts := []RegisterOption{
		WithDIMetadata("key1", "value1"),
		WithDIMetadata("key2", "value2"),
		WithDIMetadata("key1", "value1_override"),
	}

	result := MergeOptions(opts)

	if len(result.Metadata) != 2 {
		t.Errorf("MergeOptions().Metadata length = %v, want 2", len(result.Metadata))
	}

	// Last one wins for duplicates
	if result.Metadata["key1"] != "value1_override" {
		t.Errorf("MergeOptions().Metadata[\"key1\"] = %v, want value1_override", result.Metadata["key1"])
	}

	if result.Metadata["key2"] != "value2" {
		t.Errorf("MergeOptions().Metadata[\"key2\"] = %v, want value2", result.Metadata["key2"])
	}
}

func TestRegisterOption_GetAllDeps(t *testing.T) {
	tests := []struct {
		name string
		opt  RegisterOption
		want int
	}{
		{
			name: "empty",
			opt:  RegisterOption{},
			want: 0,
		},
		{
			name: "only_string_deps",
			opt:  WithDependencies("service1", "service2"),
			want: 2,
		},
		{
			name: "only_dep_specs",
			opt:  WithDeps(Eager("service1"), Lazy("service2")),
			want: 2,
		},
		{
			name: "mixed",
			opt: RegisterOption{
				Dependencies: []string{"service1", "service2"},
				Deps:         []Dep{Lazy("service3"), Optional("service4")},
			},
			want: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opt.GetAllDeps()

			if len(got) != tt.want {
				t.Errorf("GetAllDeps() length = %v, want %v", len(got), tt.want)
			}
		})
	}
}

func TestRegisterOption_GetAllDeps_StringsConvertedToEager(t *testing.T) {
	opt := WithDependencies("service1", "service2")
	deps := opt.GetAllDeps()

	if len(deps) != 2 {
		t.Fatalf("GetAllDeps() length = %v, want 2", len(deps))
	}

	for i, dep := range deps {
		if dep.Mode != DepEager {
			t.Errorf("GetAllDeps()[%d].Mode = %v, want DepEager", i, dep.Mode)
		}
	}
}

func TestRegisterOption_GetAllDepNames(t *testing.T) {
	tests := []struct {
		name string
		opt  RegisterOption
		want []string
	}{
		{
			name: "empty",
			opt:  RegisterOption{},
			want: []string{},
		},
		{
			name: "only_string_deps",
			opt:  WithDependencies("service1", "service2"),
			want: []string{"service1", "service2"},
		},
		{
			name: "only_dep_specs",
			opt:  WithDeps(Eager("service1"), Lazy("service2")),
			want: []string{"service1", "service2"},
		},
		{
			name: "mixed",
			opt: RegisterOption{
				Dependencies: []string{"service1", "service2"},
				Deps:         []Dep{Lazy("service3"), Optional("service4")},
			},
			want: []string{"service3", "service4", "service1", "service2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opt.GetAllDepNames()

			if len(got) != len(tt.want) {
				t.Errorf("GetAllDepNames() length = %v, want %v", len(got), len(tt.want))

				return
			}

			// Create a map for easier comparison (order might differ)
			wantMap := make(map[string]bool)
			for _, name := range tt.want {
				wantMap[name] = true
			}

			for _, name := range got {
				if !wantMap[name] {
					t.Errorf("GetAllDepNames() contains unexpected name: %v", name)
				}
			}
		})
	}
}

func TestRegisterOption_CombinedUsage(t *testing.T) {
	// Test a realistic combination of options
	opts := []RegisterOption{
		Scoped(),
		WithDeps(
			Eager("logger"),
			Lazy("cache"),
			Optional("metrics"),
		),
		WithDependencies("config"),
		WithDIMetadata("version", "1.0.0"),
		WithDIMetadata("author", "test"),
		WithGroup("handlers"),
	}

	result := MergeOptions(opts)

	// Verify lifecycle
	if result.Lifecycle != "scoped" {
		t.Errorf("Lifecycle = %v, want scoped", result.Lifecycle)
	}

	// Verify all dependencies
	allDeps := result.GetAllDeps()
	if len(allDeps) != 4 { // 3 from WithDeps + 1 from WithDependencies
		t.Errorf("GetAllDeps() length = %v, want 4", len(allDeps))
	}

	// Verify metadata
	if result.Metadata["version"] != "1.0.0" {
		t.Errorf("Metadata[version] = %v, want 1.0.0", result.Metadata["version"])
	}

	if result.Metadata["author"] != "test" {
		t.Errorf("Metadata[author] = %v, want test", result.Metadata["author"])
	}

	// Verify groups
	if len(result.Groups) != 1 {
		t.Errorf("Groups length = %v, want 1", len(result.Groups))
	}

	if result.Groups[0] != "handlers" {
		t.Errorf("Groups[0] = %v, want handlers", result.Groups[0])
	}
}
