package config

// Config represents the application configuration
type Config struct {
	App AppConfig `mapstructure:"app"`
}

// AppConfig represents the application-specific configuration
type AppConfig struct {
	Name  string `mapstructure:"name"`
	Debug bool   `mapstructure:"debug"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:  "spec-tdd",
			Debug: false,
		},
	}
}
