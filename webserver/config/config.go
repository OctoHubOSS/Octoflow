// Package config defined the config for the web server
package config

type Config struct {
	Token                   string                    `yaml:"token" comment:"Discord token" validate:"required"`
	PostgresURL             string                    `yaml:"postgres_url" default:"postgresql:///github" comment:"Postgres URL" validate:"required"`
	Port                    string                    `yaml:"port" default:":19318" comment:"Port to run the server on" validate:"required"`
	APIUrl                  string                    `yaml:"api_url" default:"https://v2.gitlogs.xyz" comment:"URL of the API" validate:"required"`
	DashboardInternalSecret string                    `yaml:"dashboard_internal_secret" comment:"Shared secret the Next.js dashboard uses to call /api/dashboard/* not end-user auth, just proves the caller is our own frontend"`
	GetTable                func(table string) string `yaml:"-" comment:"Function to get table names"`
}
