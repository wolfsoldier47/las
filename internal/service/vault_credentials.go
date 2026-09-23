package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"ulas-service/internal/config"
	"ulas-service/models"
)

// AAPClusterTarget describes one AAP cluster whose credentials may be
// fetched from the HCV vault at startup. Add one target per cluster.
type AAPClusterTarget struct {
	// Name identifies the cluster in logs, e.g. "solaris".
	Name string
	// AAPURL is the cluster's AAP base URL; empty means the cluster is not
	// configured and no fetch is attempted.
	AAPURL string
	// UsernameEnv / PasswordEnv are the environment variables the fetched
	// credentials are exported to, e.g. AAPUSERNAME_SOLARIS.
	UsernameEnv string
	PasswordEnv string
	// AppRole is the vault AppRole spec (role/secret/path) for this cluster.
	AppRole models.HCVAppRole
}

// BootstrapAAPCredentials fetches the AAP username/password for one cluster
// from the HCV vault using the target's AppRole spec and exports them as
// environment variables so a subsequent config.Reload() picks them up.
//
// It returns fetched=false (without error) when there is nothing to do:
// either the cluster's AAP is not configured, or its credentials are
// already provided via the environment.
func BootstrapAAPCredentials(cfg *config.AppConfig, target AAPClusterTarget) (bool, error) {
	if target.AAPURL == "" {
		return false, nil
	}
	if os.Getenv(target.UsernameEnv) != "" && os.Getenv(target.PasswordEnv) != "" {
		slog.Info("AAP credentials already set, skipping vault fetch", "cluster", target.Name)
		return false, nil
	}
	appRole := target.AppRole
	if appRole.RoleID == "" || appRole.SecretID == "" || appRole.HCVPath == "" {
		return false, fmt.Errorf("cluster %s: role_id, secret_id and hcv path are required to fetch credentials", target.Name)
	}
	if cfg.VaultAddr == "" {
		return false, fmt.Errorf("cluster %s: vault address is empty", target.Name)
	}

	cred, err := fetchVaultCredentials(cfg.VaultAddr, appRole)
	if err != nil {
		return false, fmt.Errorf("cluster %s: %w", target.Name, err)
	}

	os.Setenv(target.UsernameEnv, cred.Username)
	os.Setenv(target.PasswordEnv, cred.Password)
	slog.Info("AAP credentials fetched from vault", "cluster", target.Name, "path", appRole.HCVPath)
	return true, nil
}

// fetchVaultCredentials logs into the vault with the AppRole and reads the
// secret at the given path.
func fetchVaultCredentials(vaultAddr string, appRole models.HCVAppRole) (models.Credential, error) {
	hcv := NewHcvService()

	token, err := hcv.Token(
		strings.TrimRight(vaultAddr, "/")+"/v1/auth/approle/login",
		appRole.Namespace,
		appRole.RoleID,
		appRole.SecretID,
	)
	if err != nil {
		return models.Credential{}, fmt.Errorf("hcv approle login: %w", err)
	}
	if token == "" {
		return models.Credential{}, fmt.Errorf("hcv approle login returned an empty token")
	}

	secretURL := vaultSecretURL(vaultAddr, appRole.HCVPath, appRole.Namespace)
	resp, err := hcv.HttpClient(token).Get(secretURL)
	if err != nil {
		return models.Credential{}, fmt.Errorf("hcv secret read %s: %w", secretURL, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return models.Credential{}, fmt.Errorf("hcv secret read %s: unexpected status %s", secretURL, resp.Status)
	}

	return parseVaultCredential(resp.Bytes())
}

// vaultSecretURL joins the vault address and the configured secret path,
// ensuring the /v1/ API prefix is present exactly once.
func vaultSecretURL(vaultAddr, path string, namespace string) string {

	namespaceFix := strings.TrimPrefix(namespace, "/")
	pathFix := strings.TrimPrefix(path, "/")
	endpoint := namespaceFix + "/" + pathFix
	if !strings.HasPrefix(endpoint, "v1/") {
		endpoint = "/v1/" + endpoint
	}
	return strings.TrimRight(vaultAddr, "/") + endpoint
}

// parseVaultCredential extracts the username/password from a vault secret
// response. It supports KV v2 ({"data": {"data": {...}}}), KV v1
// ({"data": {...}}) and a raw YAML document with username_new/password_new.
func parseVaultCredential(body []byte) (models.Credential, error) {
	var wrapper struct {
		Data struct {
			Data     map[string]string `json:"data"`
			Username string            `json:"username"`
			Password string            `json:"password"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err == nil {
		if username, ok := wrapper.Data.Data["username"]; ok {
			return models.Credential{
				Username: username,
				Password: wrapper.Data.Data["password"],
			}, nil
		}
		if wrapper.Data.Username != "" {
			return models.Credential{
				Username: wrapper.Data.Username,
				Password: wrapper.Data.Password,
			}, nil
		}
	}

	// Fallback: the secret payload itself may be a YAML document.
	var cred models.Credential
	if err := yaml.Unmarshal(body, &cred); err != nil {
		return models.Credential{}, fmt.Errorf("unable to parse vault secret payload: %w", err)
	}
	if cred.Username == "" {
		return models.Credential{}, fmt.Errorf("vault secret payload did not contain username_new")
	}
	return cred, nil
}
