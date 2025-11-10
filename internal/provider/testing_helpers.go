// Package provider contains test helper functions shared across test files
package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/aaearon/terraform-provider-cyberarksia/internal/client"
)

// getProviderDataFromEnv creates a provider data instance from environment variables
// for use in CheckDestroy functions to verify resources were deleted from the API
func getProviderDataFromEnv() (*ProviderData, error) {
	username := os.Getenv("CYBERARK_USERNAME")
	password := os.Getenv("CYBERARK_PASSWORD")

	if username == "" || password == "" {
		return nil, fmt.Errorf("CYBERARK_USERNAME and CYBERARK_PASSWORD must be set")
	}

	// Create authentication context
	authConfig := &client.AuthConfig{
		Username:    username,
		Password:    password,
		IdentityURL: os.Getenv("CYBERARK_IDENTITY_URL"), // Optional
	}

	authCtx, err := client.NewISPAuth(context.Background(), authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	// Create SIA API client
	siaAPI, err := client.NewSIAClient(context.Background(), authCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create SIA client: %w", err)
	}

	return &ProviderData{
		SIAAPI:      siaAPI,
		AuthContext: authCtx,
	}, nil
}
