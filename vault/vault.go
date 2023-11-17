package vault

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"github.com/joomcode/errorx"
	"github.com/sovamorco/gommon/config"
)

const (
	CredsEnvVar  = "VAULT_CONFIG" //nolint:gosec
	RenewTimeout = 60 * time.Second
	RenewBuffer  = 5 * time.Minute
)

var ErrNoAuth = errors.New("at least one auth method should be specified")

type AppRoleConfig struct {
	RoleID   string `mapstructure:"role_id"`
	SecretID string `mapstructure:"secret_id"`
}

type Creds struct {
	Host       string        `mapstructure:"host"`
	Method     string        `mapstructure:"method"`
	Parameters AppRoleConfig `mapstructure:"parameters"`
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

	if creds.Method != "approle" {
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
			RoleId:   creds.Parameters.RoleID,
			SecretId: creds.Parameters.SecretID,
		},
	)
	if err != nil {
		return nil, errorx.Decorate(err, "authenticate")
	}

	err = vc.SetToken(auth.Auth.ClientToken)
	if err != nil {
		return nil, errorx.Decorate(err, "set token")
	}

	go renew(ctx, vc, time.Duration(auth.Auth.LeaseDuration)*time.Second)

	return vc, nil
}

func renew(ctx context.Context, vc *vault.Client, leaseDur time.Duration) {
	ctx = context.WithoutCancel(ctx)

	time.Sleep(leaseDur - RenewBuffer)

	ctx, cancel := context.WithTimeout(ctx, RenewTimeout)
	defer cancel()

	a, err := vc.Auth.TokenRenewSelf(ctx, schema.TokenRenewSelfRequest{}) //nolint:exhaustruct
	if err != nil {
		return
	}

	go renew(ctx, vc, time.Duration(a.Auth.LeaseDuration)*time.Second)
}
