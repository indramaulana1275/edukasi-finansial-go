package services

import (
	"sync"

	"github.com/casbin/casbin/v2"
	"github.com/goravel/framework/support/path"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

var (
	enforcerOnce sync.Once
	enforcer     *casbin.Enforcer
	enforcerErr  error
)

// Enforcer lazily builds a Casbin enforcer from the model and policy files
// configured under the "casbin" key (see config/casbin.go).
func Enforcer() (*casbin.Enforcer, error) {
	enforcerOnce.Do(func() {
		modelFile := facades.Config().GetString("casbin.model", "casbin_model.conf")
		policyFile := facades.Config().GetString("casbin.policy", "casbin_policy.csv")

		enforcer, enforcerErr = casbin.NewEnforcer(path.Config(modelFile), path.Config(policyFile))
	})

	return enforcer, enforcerErr
}
