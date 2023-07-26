package vault

import (
	"context"
	"errors"
	"os"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"github.com/joomcode/errorx"
	"github.com/sovamorco/gommon/config"
)

const (
	CredsEnvVar = "VAULT_CONFIG" //nolint:gosec
)

var ErrNoAuth = errors.New("at least one auth method should be specified")

type AppRoleConfig struct {
	RoleID   string `mapstructure:"role_id"`
	SecretID string `mapstructure:"secret_id"`
}

type Creds struct {
	Host    string         `mapstructure:"host"`
	AppRole *AppRoleConfig `mapstructure:"approle"`
}

func ClientFromEnv(ctx context.Context) (*vault.Client, error) {
	credsPath := os.Getenv(CredsEnvVar)
	if credsPath == "" {
		return nil, config.MissingEnvError{EnvVarName: CredsEnvVar}
	}

	var creds Creds

	err := config.LoadConfig(ctx, credsPath, &creds)
	if err != nil {
		return nil, errorx.Decorate(err, "load vault credentials")
	}

	if creds.AppRole == nil {
		return nil, ErrNoAuth
	}

	vc, err := vault.New(
		vault.WithAddress(creds.Host),
	)
	if err != nil {
		return nil, errorx.Decorate(err, "create vault client")
	}

	auth, err := vc.Auth.AppRoleLogin(ctx,
		schema.AppRoleLoginRequest{
			RoleId:   creds.AppRole.RoleID,
			SecretId: creds.AppRole.SecretID,
		},
	)
	if err != nil {
		return nil, errorx.Decorate(err, "authenticate")
	}

	err = vc.SetToken(auth.Auth.ClientToken)
	if err != nil {
		return nil, errorx.Decorate(err, "set token")
	}

	return vc, nil
}
