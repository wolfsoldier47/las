package models

import "ulas-service/internal/client/httpi"

type Auth struct {
	ClientToken string `json:"client_token"`
}

type AppRoleResponse struct {
	Auth Auth `json:"auth"`
}

type AppRoleRequest struct {
	RoleId   string `json:"role_id"`
	SecretId string `json:"secret_id"`
}

type Credential struct {
	Username string `yaml:"username_new"`
	Password string `yaml:"password_new"`
}

type HCVAppRole struct {
	RoleID    string
	SecretID  string
	HCVPath   string
	Namespace string
}

type HcvService interface {
	Token(url string, namespace string, roleId string, secretId string) (string, error)
	HttpClient(token string) httpi.Client
}
