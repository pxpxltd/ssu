package config

// Source identifies which configuration layer set a value.
type Source string

const (
	// SourceDefault means the value comes from built-in defaults.
	SourceDefault Source = "default"

	// SourceGlobal means the value was set in ~/.ssu/config.yaml.
	SourceGlobal Source = "global (~/.ssu/config.yaml)"

	// SourceProject means the value was set in .ssu.yaml.
	SourceProject Source = "project (.ssu.yaml)"

	// SourceEnv means the value was set via an SSU_ environment variable.
	SourceEnv Source = "env"

	// SourceFlag means the value was set via a CLI flag.
	SourceFlag Source = "flag"
)

// AnnotatedValue pairs a configuration value with the source that set it.
type AnnotatedValue struct {
	Value  any
	Source Source
}

// AnnotatedConfig tracks which layer set each configuration key.
// Keys use flat dot-notation (e.g., "git.parallel_jobs").
type AnnotatedConfig struct {
	values map[string]AnnotatedValue
}

// NewAnnotatedConfig creates an empty AnnotatedConfig.
func NewAnnotatedConfig() *AnnotatedConfig {
	return &AnnotatedConfig{values: make(map[string]AnnotatedValue)}
}

// Set records a value and its source for a config key.
func (ac *AnnotatedConfig) Set(key string, value any, source Source) {
	ac.values[key] = AnnotatedValue{Value: value, Source: source}
}

// Get returns the annotated value for a config key.
// If the key is not found, it returns an AnnotatedValue with nil Value and SourceDefault.
func (ac *AnnotatedConfig) Get(key string) AnnotatedValue {
	if v, ok := ac.values[key]; ok {
		return v
	}
	return AnnotatedValue{Source: SourceDefault}
}

// Keys returns all tracked config keys in no particular order.
func (ac *AnnotatedConfig) Keys() []string {
	keys := make([]string, 0, len(ac.values))
	for k := range ac.values {
		keys = append(keys, k)
	}
	return keys
}
