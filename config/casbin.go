package config

import (
	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

func init() {
	config := facades.Config()
	config.Add("casbin", map[string]any{
		// Casbin model file, relative to the config directory.
		"model": config.Env("CASBIN_MODEL", "casbin_model.conf"),

		// Casbin policy file, relative to the config directory.
		"policy": config.Env("CASBIN_POLICY", "casbin_policy.csv"),
	})
}
