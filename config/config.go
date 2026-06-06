package config

import "time"

type ApplicationConfig struct {
	Database       map[string]DatabaseConfig `json:"database"`
	Port           PortConfig                `json:"port"`
	NewRelicCfg    NewRelicConfig            `json:"newrelic"`
	Log            Log                       `json:"log"`
	ContextTimeout int                       `json:"context_timeout"`
}

type PortConfig struct {
	Host            string        `json:"host"`
	Service         int           `json:"service"`
	Profiler        int           `json:"profiler"`
	ServiceTimeout  time.Duration `json:"servicetimeout"`
	ProfilerTimeout time.Duration `json:"profilertimeout"`
}

type DatabaseConfig struct {
	Username        string        `json:"username"`
	Password        string        `json:"password"`
	Hostname        string        `json:"hostname"`
	DBname          string        `json:"dbname"`
	SSLMode         string        `json:"sslmode"`
	Timeout         time.Duration `json:"timeout"`
	MaxOpenConns    int           `json:"maxopenconns"`
	MaxIdleConns    int           `json:"maxidleconns"`
	ConnMaxLifetime time.Duration `json:"connmaxlifetime"`
}

type NewRelicConfig struct {
	IsActive   bool   `json:"isActive"`
	Name       string `json:"name"`
	LicenseKey string `json:"license_key"`
	IngestKey  string `json:"ingest_key"`
	Region     string `json:"region"`
}

type Log struct {
	Request         bool `json:"request"`
	SuccessResponse bool `json:"success_response"`
	FailedResponse  bool `json:"failed_response"`
}
