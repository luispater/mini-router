package models

import (
	"time"
)

type APIKey struct {
	// ID is the primary key
	ID uint `json:"id" yaml:"id"`

	// Key is the encrypted API key
	Key string `json:"key" yaml:"key"`
	// Name is the name of the API key
	Name string `json:"name" yaml:"name"`
	// ExpiresAt is the expiration time
	ExpiresAt time.Time `json:"expires_at" yaml:"expires_at"`
	// IsActive indicates whether the key is active
	IsActive bool `json:"is_active" yaml:"is_active"`

	// RPS is the requests per second, if 0, use the model's default value
	RPS int `json:"rps" yaml:"rps"`
	// TPS is the tokens per second, if 0, use the model's default value
	TPS int `json:"tps" yaml:"tps"`
	// RPM is the requests per minute, if 0, use the model's default value
	RPM int `json:"rpm" yaml:"rpm"`
	// TPM is the tokens per minute, if 0, use the model's default value
	TPM int `json:"tpm" yaml:"tpm"`
	// RPH is the requests per hour, if 0, use the model's default value
	RPH int `json:"rph" yaml:"rph"`
	// TPH is the tokens per hour, if 0, use the model's default value
	TPH int `json:"tph" yaml:"tph"`
	// RPD is the requests per day, if 0, use the model's default value
	RPD int `json:"rpd" yaml:"rpd"`
	// TPD is the tokens per day, if 0, use the model's default value
	TPD int `json:"tpd" yaml:"tpd"`
}
