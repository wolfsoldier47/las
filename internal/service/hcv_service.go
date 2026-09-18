package service

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
	"ulas-service/internal/client/httpi"
	"ulas-service/models"
)

var onceHcvService sync.Once

type hcvService struct {
}

var instanceHcvService *hcvService

func NewHcvService() models.HcvService {
	onceHcvService.Do(func() {
		instanceHcvService = &hcvService{}
	})
	return instanceHcvService
}

func (h hcvService) Token(url string, namespace string, roleId string, secretId string) (string, error) {
	var appRoleResponse = models.AppRoleResponse{}
	appRole := models.AppRoleRequest{
		RoleId:   roleId,
		SecretId: secretId,
	}
	body, _ := json.Marshal(appRole)

	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("X-Vault-Namespace", namespace)
	request.Header.Set("Connection", "close")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	response, err := client.Do(request)

	defer response.Body.Close()

	if err != nil {
		return "", err
	}
	body, _ = io.ReadAll(response.Body)
	err = json.Unmarshal(body, &appRoleResponse)
	if err != nil {
		return "", err
	}

	return appRoleResponse.Auth.ClientToken, err
}

func (h hcvService) HttpClient(token string) httpi.Client {
	headers := make(http.Header)
	headers.Set("X-Vault-Token", token)
	headers.Set("Connection", "close")

	client := httpi.NewBuilder().
		SetHeaders(headers).
		SetConnectionTimeout(30 * time.Second).
		SetResponseTimeout(30 * time.Second).
		SetUserAgent("ulas-service").
		Build()

	return client
}
