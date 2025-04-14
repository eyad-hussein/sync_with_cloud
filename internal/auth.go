package internal

import (
	"context"
	"net/http"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

type AuthClient struct {
	credentialsPath string
}

func NewAuthClient(credentialsPath string) *AuthClient {
	return &AuthClient{credentialsPath: credentialsPath}
}

func (a *AuthClient) GetAuthClient() (*http.Client, error) {
	b, err := os.ReadFile(a.credentialsPath)

	if err != nil {
		return &http.Client{}, err
	}

	config, err := google.JWTConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return &http.Client{}, err
	}

	return config.Client(context.Background()), nil
}
