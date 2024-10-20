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
	Method     string         `mapstructure:"method"`
	Parameters *AppRoleConfig `mapstructure:"parameters"`
}

func approleLogin(ctx context.Context, vc *vault.Client, creds *Creds) error {
	auth, err := vc.Auth.AppRoleLogin(ctx,
		schema.AppRoleLoginRequest{
			RoleId:   creds.Parameters.RoleID,
			SecretId: creds.Parameters.SecretID,
		},
	)
	if err != nil {
		return errorx.Decorate(err, "authenticate")
	}

	err = vc.SetToken(auth.Auth.ClientToken)
	if err != nil {
		return errorx.Decorate(err, "set token")
	}

	go renew(ctx, vc, time.Duration(auth.Auth.LeaseDuration)*time.Second)

	return nil
}

func workloadLogin() error {
	return nil
}

func getWorkloadIdentityConfig() *Creds {
	return &Creds{
		Method:     "workload",
		Parameters: nil,
	}
}

func ClientFromEnv(ctx context.Context) (*vault.Client, error) {
	var creds *Creds

	var err error

	credsPath := os.Getenv(CredsEnvVar)
	if credsPath == "" {
		creds = getWorkloadIdentityConfig()
	} else {
		err = config.LoadConfig(ctx, credsPath, &creds)
		if err != nil {
			return nil, errorx.Decorate(err, "load vault credentials")
		}
	}

	vc, err := vault.New(
		vault.WithEnvironment(),
	)
	if err != nil {
		return nil, errorx.Decorate(err, "create vault client")
	}

	switch creds.Method {
	case "approle":
		err = approleLogin(ctx, vc, creds)
	case "workload":
		err = workloadLogin()
	default:
		err = ErrNoAuth
	}

	if err != nil {
		return nil, errorx.Decorate(err, "error logging in with method %s", creds.Method)
	}

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
