// Package client provides CyberArk SIA API client wrappers
package client

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cyberark/ark-sdk-golang/pkg/common"
	"github.com/cyberark/ark-sdk-golang/pkg/common/isp"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mitchellh/mapstructure"
)

const (
	// certificatesURL is the base endpoint for certificate operations (✅ VALIDATED via API testing)
	certificatesURL = "/api/certificates"
	// certificateURL is the endpoint for individual certificate operations (✅ VALIDATED via API testing)
	certificateURL = "/api/certificates/%s"
)

// CertificatesClient handles certificate CRUD operations using ARK SDK's ISP client.
// Follows the exact pattern from WorkspacesDB service for authentication and HTTP operations.
//
// Authentication:
//   - Uses SDK's isp.ArkISPServiceClient for REST API calls
//   - Token access via ispAuth.Token.Token (direct field access)
//   - URL construction auto-handled by isp.FromISPAuth()
//   - Token refresh handled automatically via callback
//
// Reference: /pkg/services/sia/workspaces/db/ark_sia_workspaces_db_service.go
type CertificatesClient struct {
	authCtx *ISPAuthContext          // Provider's authentication context (in-memory profile)
	client  *isp.ArkISPServiceClient // SDK's authenticated HTTP client
}

// NewCertificatesClient creates a new certificates client using ARK SDK authentication.
// Follows WorkspacesDB pattern (lines 32-53).
//
// The SDK handles:
//   - Base URL construction (https://{subdomain}.dpa.{domain})
//   - Authorization headers (Bearer token)
//   - Token refresh (15-min JWT lifecycle)
//   - All required HTTP headers
//
// Parameters:
//   - authCtx: *ISPAuthContext from provider configuration (in-memory profile)
//
// Returns:
//   - *CertificatesClient: Initialized client ready for CRUD operations
//   - error: If client initialization fails
func NewCertificatesClient(ctx context.Context, authCtx *ISPAuthContext) (*CertificatesClient, error) {
	tflog.Debug(ctx, "Initializing Certificates API client", map[string]interface{}{
		"service": "certificates",
	})

	certsClient := &CertificatesClient{
		authCtx: authCtx,
	}

	// Create ISP service client (SAME as WorkspacesDB line 45)
	// This auto-constructs the base URL and configures authentication
	client, err := isp.FromISPAuth(
		authCtx.ISPAuth,            // Use ISPAuth from context
		"dpa",                      // Service name (constructs https://{subdomain}.dpa.{domain})
		".",                        // Separator character
		"",                         // Base path (empty for root API)
		certsClient.refreshSIAAuth, // Token refresh callback
	)
	if err != nil {
		tflog.Error(ctx, "Failed to initialize Certificates API client", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to create ISP client for certificates: %w", err)
	}

	certsClient.client = client
	tflog.Info(ctx, "Certificates API client initialized successfully")
	return certsClient, nil
}

// refreshSIAAuth refreshes the authentication token when it expires.
// Called automatically by SDK when token approaches 15-min expiration.
// CRITICAL: Re-authenticates with in-memory profile to bypass cache
//
// Parameters:
//   - client: *common.ArkClient to refresh
//
// Returns:
//   - error: If token refresh fails
func (c *CertificatesClient) refreshSIAAuth(client *common.ArkClient) error {
	// Note: No context available in SDK callback, can't use tflog here
	// TODO: Add logging when SDK provides context support

	// Re-authenticate with in-memory profile (force=true to bypass cache)
	_, err := c.authCtx.ISPAuth.Authenticate(
		c.authCtx.Profile,     // In-memory profile (NOT nil)
		c.authCtx.AuthProfile, // Auth profile
		c.authCtx.Secret,      // Secret
		true,                  // force=true (bypass cache)
		false,                 // refreshAuth=false
	)
	if err != nil {
		return fmt.Errorf("failed to refresh authentication: %w", err)
	}

	// Refresh the client with new token
	return isp.RefreshClient(client, c.authCtx.ISPAuth)
}

// Certificate represents a TLS/SSL certificate stored in SIA.
// Maps to API response from GET /api/certificates/{id}.
// All field names use snake_case matching API contract (✅ VALIDATED via API testing).
//
// Field Tags:
//   - json: snake_case (matches API)
//   - mapstructure: snake_case (for SDK deserialization)
type Certificate struct {
	Labels          map[string]string    `json:"labels" mapstructure:"labels"`
	Metadata        *CertificateMetadata `json:"metadata" mapstructure:"metadata"`
	CertificateID   string               `json:"certificate_id" mapstructure:"certificate_id"`
	TenantID        string               `json:"tenant_id" mapstructure:"tenant_id"`
	CertName        string               `json:"cert_name" mapstructure:"cert_name"`
	CertBody        string               `json:"cert_body" mapstructure:"cert_body"`
	CertDescription string               `json:"cert_description" mapstructure:"cert_description"`
	CertType        string               `json:"cert_type" mapstructure:"cert_type"`
	DomainName      string               `json:"domain_name" mapstructure:"domain_name"`
	ExpirationDate  string               `json:"expiration_date" mapstructure:"expiration_date"`
}

// CertificateMetadata contains X.509 certificate metadata extracted by SIA.
// Nested within Certificate response from API.
type CertificateMetadata struct {
	Issuer                 string   `json:"issuer" mapstructure:"issuer"`                                     // Certificate issuer DN
	Subject                string   `json:"subject" mapstructure:"subject"`                                   // Certificate subject DN
	ValidFrom              string   `json:"valid_from" mapstructure:"valid_from"`                             // Validity start (Unix timestamp string)
	ValidTo                string   `json:"valid_to" mapstructure:"valid_to"`                                 // Validity end (Unix timestamp string)
	SerialNumber           string   `json:"serial_number" mapstructure:"serial_number"`                       // Certificate serial number (decimal format string)
	SubjectAlternativeName []string `json:"subject_alternative_name" mapstructure:"subject_alternative_name"` // SANs (empty array if none)
}

// CertificateCreateRequest represents the payload for POST /api/certificates.
// All fields use snake_case matching API contract (✅ VALIDATED via API testing).
//
// Required Fields:
//   - CertBody: PEM encoded certificate content
//
// Optional Fields:
//   - CertName, CertDescription, CertType, DomainName, Labels
type CertificateCreateRequest struct {
	Labels          map[string]string `json:"labels,omitempty" mapstructure:"labels"`
	CertName        string            `json:"cert_name,omitempty" mapstructure:"cert_name"`
	CertBody        string            `json:"cert_body" mapstructure:"cert_body"`
	CertDescription string            `json:"cert_description,omitempty" mapstructure:"cert_description"`
	CertType        string            `json:"cert_type,omitempty" mapstructure:"cert_type"`
	DomainName      string            `json:"domain_name,omitempty" mapstructure:"domain_name"`
}

// CertificateUpdateRequest represents the payload for PUT /api/certificates/{id}.
// ⚠️ CRITICAL: cert_body is REQUIRED for ALL updates (✅ VALIDATED via API testing - Issue #4).
//
// Even metadata-only updates require cert_body from state.
// Attempting update without cert_body returns 400 Bad Request.
//
// All fields use snake_case matching API contract.
type CertificateUpdateRequest struct {
	Labels          map[string]string `json:"labels,omitempty" mapstructure:"labels"`
	CertName        string            `json:"cert_name,omitempty" mapstructure:"cert_name"`
	CertBody        string            `json:"cert_body" mapstructure:"cert_body"`
	CertDescription string            `json:"cert_description,omitempty" mapstructure:"cert_description"`
	CertType        string            `json:"cert_type,omitempty" mapstructure:"cert_type"`
	DomainName      string            `json:"domain_name,omitempty" mapstructure:"domain_name"`
}

// CertificateListResponse represents the response from GET /api/certificates.
// ⚠️ CRITICAL: Response structure is NESTED (✅ VALIDATED via API testing - Issue #9).
//
// Structure: {tenant_id, certificates: {items: [...]}}
// Field names DIFFER from GET /api/certificates/{id} (Issue #15):
//   - LIST: "body" → GET: "cert_body"
//   - LIST: "domain" → GET: "domain_name"
//
// Data sources must map field names when using LIST endpoint.
type CertificateListResponse struct {
	Certificates *CertificateListContainer `json:"certificates" mapstructure:"certificates"`
	TenantID     string                    `json:"tenant_id" mapstructure:"tenant_id"`
}

// CertificateListContainer wraps the array of certificates in LIST response.
type CertificateListContainer struct {
	Items []CertificateListItem `json:"items" mapstructure:"items"`
}

// CertificateListItem represents a single certificate in LIST response.
// ⚠️ Field names differ from Certificate struct (Issue #15):
//   - "body" instead of "cert_body"
//   - "domain" instead of "domain_name"
type CertificateListItem struct {
	CertificateID   string               `json:"certificate_id" mapstructure:"certificate_id"`
	Body            string               `json:"body" mapstructure:"body"`     // ⚠️ Maps to cert_body
	Domain          string               `json:"domain" mapstructure:"domain"` // ⚠️ Maps to domain_name
	CertName        string               `json:"cert_name" mapstructure:"cert_name"`
	CertDescription string               `json:"cert_description" mapstructure:"cert_description"`
	Metadata        *CertificateMetadata `json:"metadata" mapstructure:"metadata"`
	Labels          map[string]string    `json:"labels" mapstructure:"labels"`
	ExpirationDate  string               `json:"expiration_date" mapstructure:"expiration_date"`
}

// ValidatePEMCertificate validates PEM-encoded certificate content.
// Performs client-side validation before API call to provide immediate feedback.
//
// Validation Steps:
//  1. PEM format check (must contain "-----BEGIN CERTIFICATE-----")
//  2. PEM decode to ASN.1 DER
//  3. X.509 parse (must be valid certificate structure)
//  4. Private key check (MUST NOT contain private key material)
//  5. ❌ NO EXPIRATION CHECK (deferred to API per data-model.md Issue #13)
//
// Rationale for skipping expiration check:
//   - SIA permits uploading near-expired/expired certs for staging rotations
//   - Client-side rejection blocks legitimate workflows (testing, audit)
//   - Let SIA API enforce its own expiration policies
//
// Parameters:
//   - pemData: PEM-encoded certificate string (with newlines)
//
// Returns:
//   - error: Validation failure with specific reason
//   - nil: Certificate is valid
func ValidatePEMCertificate(pemData string) error {
	// 1. PEM format check
	if !strings.Contains(pemData, "-----BEGIN CERTIFICATE-----") {
		return fmt.Errorf("invalid PEM format: must contain '-----BEGIN CERTIFICATE-----' header")
	}

	// 2. PEM decode
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return fmt.Errorf("failed to decode PEM block: invalid PEM structure")
	}

	// 3. Check PEM block type (must be CERTIFICATE, not PRIVATE KEY)
	if block.Type != "CERTIFICATE" {
		// Check for common private key types
		if strings.Contains(block.Type, "PRIVATE KEY") {
			return fmt.Errorf("certificate content must NOT contain private key material (found: %s)", block.Type)
		}
		return fmt.Errorf("PEM block is not a certificate (type: %s, expected: CERTIFICATE)", block.Type)
	}

	// 4. X.509 parse (verify valid certificate structure)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse X.509 certificate: %w", err)
	}

	// 5. ❌ SKIP expiration check (defer to SIA API - Issue #13)
	// Rationale: SIA may accept expired certs for staging/testing/audit
	// Previous check (REMOVED):
	// now := time.Now()
	// if now.Before(cert.NotBefore) || now.After(cert.NotAfter) { ... }

	// Basic sanity check: cert object was created successfully
	_ = cert // Use cert variable to avoid unused error

	return nil
}

// CreateCertificate creates a new certificate in SIA.
// Follows WorkspacesDB.AddDatabase pattern (lines 113-173).
//
// API Endpoint: POST /api/certificates
// Success Response: HTTP 201 Created with full Certificate object (8 fields)
// ✅ VALIDATED: CREATE returns FULL object (not ID-only) - Issue #6 confirmed
//
// Response Fields (8 fields from CREATE):
//   - certificate_id, tenant_id, cert_body, cert_name, cert_description
//   - domain_name, expiration_date, labels
//
// Missing from CREATE (available via GET):
//   - metadata, checksum, version, created_by, last_updated_by, updated_time
//
// Possible Errors:
//   - 400 Bad Request: Invalid certificate content or missing cert_body
//   - 409 Conflict: Duplicate certificate name
//   - 401 Unauthorized: Authentication failure
//
// Parameters:
//   - ctx: Context for request cancellation and tracing
//   - req: *CertificateCreateRequest containing certificate data (cert_body required)
//
// Returns:
//   - *Certificate: Created certificate with all computed fields from CREATE response
//   - error: API error or network failure (wrapped for retry/mapping)
func (c *CertificatesClient) CreateCertificate(ctx context.Context, req *CertificateCreateRequest) (*Certificate, error) {
	// Validate required field
	if req.CertBody == "" {
		return nil, fmt.Errorf("cert_body is required for certificate creation")
	}

	// Convert request to map for JSON serialization (mapstructure pattern)
	var requestMap map[string]interface{}
	if err := mapstructure.Decode(req, &requestMap); err != nil {
		return nil, fmt.Errorf("failed to encode CREATE request: %w", err)
	}

	// Execute POST request with retry logic
	var cert Certificate
	err := RetryWithBackoff(ctx, &RetryConfig{
		MaxRetries: DefaultMaxRetries,
		BaseDelay:  BaseDelay,
		MaxDelay:   MaxDelay,
	}, func() error {
		// POST request using SDK client (auto-handles auth headers)
		response, postErr := c.client.Post(ctx, certificatesURL, requestMap)
		if postErr != nil {
			return postErr
		}
		defer response.Body.Close()

		// Read response body for processing
		bodyBytes, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return fmt.Errorf("failed to read response body: %w", readErr)
		}

		// Check HTTP status code
		if response.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed to create certificate - [%d] - [%s]",
				response.StatusCode, string(bodyBytes))
		}

		// Deserialize JSON response from bodyBytes (SAME as WorkspacesDB line 162-165)
		certJSON, err := common.DeserializeJSONSnake(io.NopCloser(bytes.NewReader(bodyBytes)))
		if err != nil {
			return fmt.Errorf("failed to deserialize CREATE response: %w", err)
		}

		// Convert to Certificate struct
		if err := mapstructure.Decode(certJSON, &cert); err != nil {
			return fmt.Errorf("failed to decode CREATE response: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &cert, nil
}

// GetCertificate retrieves a certificate by ID from SIA.
// Follows WorkspacesDB.Database pattern (lines 302-339).
//
// API Endpoint: GET /api/certificates/{id}
// Success Response: HTTP 200 OK with full Certificate object (14 fields)
//
// Response Fields (14 fields from GET):
//   - certificate_id, tenant_id, cert_body, cert_name, cert_description
//   - domain_name, expiration_date, labels, checksum, version
//   - created_by, last_updated_by, updated_time, metadata (nested)
//
// ✅ CRITICAL FINDINGS:
//   - cert_body IS returned by GET (not write-only) - must store in state with Sensitive=true
//   - metadata is nested object with 6 fields (Issue #3 confirmed)
//   - Drift detection fields: version, checksum, updated_time (Issue #14 confirmed)
//
// Possible Errors:
//   - 404 Not Found: Certificate doesn't exist (returns nil, no error for drift handling)
//   - 401 Unauthorized: Authentication failure
//   - 403 Forbidden: Permission denied
//
// Parameters:
//   - ctx: Context for request cancellation and tracing
//   - id: Certificate ID (string, e.g., "1761251731882561")
//
// Returns:
//   - *Certificate: Certificate with all fields, or nil if not found (404)
//   - error: API error (nil for 404 - drift detection pattern)
func (c *CertificatesClient) GetCertificate(ctx context.Context, id string) (*Certificate, error) {
	if id == "" {
		return nil, fmt.Errorf("certificate ID cannot be empty")
	}

	// Construct endpoint URL
	url := fmt.Sprintf(certificateURL, id)

	// Execute GET request
	response, err := c.client.Get(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate %s: %w", id, err)
	}
	defer response.Body.Close()

	// Handle HTTP status codes
	if response.StatusCode == http.StatusNotFound {
		// Return nil for drift detection (resource removed externally)
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get certificate %s - [%d] - [%s]",
			id, response.StatusCode, common.SerializeResponseToJSON(response.Body))
	}

	// Deserialize JSON response (SAME as WorkspacesDB line 324-327)
	certJSON, err := common.DeserializeJSONSnake(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize GET response: %w", err)
	}

	// Convert to Certificate struct
	var cert Certificate
	if err := mapstructure.Decode(certJSON, &cert); err != nil {
		return nil, fmt.Errorf("failed to decode GET response: %w", err)
	}

	return &cert, nil
}

// UpdateCertificate updates an existing certificate in SIA.
// Follows WorkspacesDB.UpdateDatabase pattern (database_workspace_resource.go lines 418-523).
//
// ⚠️ CRITICAL REQUIREMENT (Issue #4):
//   - cert_body is REQUIRED for ALL updates (even metadata-only changes)
//   - API returns 400 Bad Request if cert_body is missing
//   - Provider MUST persist cert_body in state and include in all update requests
//
// API Endpoint: PUT /api/certificates/{id}
// Success Response: HTTP 200 OK with full Certificate object (8 fields)
// Possible Errors:
//   - 400 Bad Request: Missing cert_body or invalid certificate content
//   - 404 Not Found: Certificate doesn't exist
//   - 409 Conflict: Duplicate name (if renaming)
//
// Parameters:
//   - ctx: Context for request cancellation and tracing
//   - id: Certificate ID (string, e.g., "1761251731882561")
//   - req: *CertificateUpdateRequest containing all certificate fields (cert_body REQUIRED)
//
// Returns:
//   - *Certificate: Updated certificate with all computed fields
//   - error: API error or network failure (wrapped for retry/mapping)
func (c *CertificatesClient) UpdateCertificate(ctx context.Context, id string, req *CertificateUpdateRequest) (*Certificate, error) {
	// Validate required field (cert_body)
	if req.CertBody == "" {
		return nil, fmt.Errorf("cert_body is required for all certificate updates (API requirement)")
	}

	// Construct endpoint URL
	url := fmt.Sprintf(certificateURL, id)

	// Convert request to map for JSON serialization
	var requestMap map[string]interface{}
	if err := mapstructure.Decode(req, &requestMap); err != nil {
		return nil, fmt.Errorf("failed to encode UPDATE request: %w", err)
	}

	// Execute PUT request with retry logic
	var cert Certificate
	err := RetryWithBackoff(ctx, &RetryConfig{
		MaxRetries: DefaultMaxRetries,
		BaseDelay:  BaseDelay,
		MaxDelay:   MaxDelay,
	}, func() error {
		// PUT request using SDK client (auto-handles auth headers)
		response, putErr := c.client.Put(ctx, url, requestMap)
		if putErr != nil {
			return putErr
		}
		defer response.Body.Close()

		// Check HTTP status code
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to update certificate - [%d] - [%s]",
				response.StatusCode, common.SerializeResponseToJSON(response.Body))
		}

		// Deserialize JSON response (SAME as WorkspacesDB line 287-290)
		certJSON, err := common.DeserializeJSONSnake(response.Body)
		if err != nil {
			return fmt.Errorf("failed to deserialize UPDATE response: %w", err)
		}

		// Convert to Certificate struct
		if err := mapstructure.Decode(certJSON, &cert); err != nil {
			return fmt.Errorf("failed to decode UPDATE response: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &cert, nil
}

// DeleteCertificate deletes a certificate from SIA by ID.
// Follows WorkspacesDB.DeleteDatabase pattern (lines 176-203).
//
// HTTP DELETE /api/certificates/{id}
//
// Success Responses:
//   - 200 OK: Certificate deleted successfully
//   - 204 No Content: Certificate deleted successfully
//   - 404 Not Found: Treated as success (certificate already deleted)
//
// Error Responses:
//   - 409 Conflict: CERTIFICATE_IN_USE - Certificate referenced by database workspaces
//   - 401 Unauthorized: Authentication failure
//   - 403 Forbidden: Permission denied
//
// Delete Protection:
//
//	If certificate is referenced by database workspaces, API returns 409 Conflict
//	with list of dependent workspace IDs. Provider MUST map this to actionable error.
//
// Parameters:
//   - ctx: context.Context for cancellation and timeout
//   - certificateID: Certificate ID to delete (e.g., "1761251731882561")
//
// Returns:
//   - error: nil on success (including 404), error on failure (especially 409)
func (c *CertificatesClient) DeleteCertificate(ctx context.Context, certificateID string) error {
	if certificateID == "" {
		return fmt.Errorf("certificate ID cannot be empty")
	}

	endpoint := fmt.Sprintf(certificateURL, certificateID)

	// Execute DELETE request
	// NOTE: SDK bug - passing nil causes panic. Pass empty map as workaround.
	response, err := c.client.Delete(ctx, endpoint, map[string]string{})
	if err != nil {
		return fmt.Errorf("failed to delete certificate %s: %w", certificateID, err)
	}
	defer response.Body.Close()

	// Handle HTTP status codes (SAME as WorkspacesDB line 191-198)
	if response.StatusCode == http.StatusNotFound {
		// Treat as success (already deleted)
		return nil
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete certificate %s - [%d] - [%s]",
			certificateID, response.StatusCode, common.SerializeResponseToJSON(response.Body))
	}

	return nil
}

// ListCertificates retrieves all certificates from SIA.
// Follows WorkspacesDB.listDatabasesWithFilters pattern (lines 79-110).
//
// HTTP GET /api/certificates
//
// Response Structure (✅ VALIDATED - Issue #9):
//
//	{
//	  "tenant_id": "...",
//	  "certificates": {
//	    "items": [...]
//	  }
//	}
//
// Field Name Differences (✅ VALIDATED - Issue #15):
//   - LIST: "body" → GET: "cert_body" ⚠️ MUST MAP
//   - LIST: "domain" → GET: "domain_name" ⚠️ MUST MAP
//   - Other fields: Same names as GET
//
// Usage: Data source implementation (data "cyberark_sia_certificate" lookup by name)
//
// Parameters:
//   - ctx: context.Context for cancellation and timeout
//
// Returns:
//   - []CertificateListItem: Array of certificates (empty if none exist)
//   - error: nil on success, error on failure
func (c *CertificatesClient) ListCertificates(ctx context.Context) ([]CertificateListItem, error) {
	// Execute GET request (SAME as WorkspacesDB line 89)
	httpResponse, err := c.client.Get(ctx, certificatesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates: %w", err)
	}
	defer httpResponse.Body.Close()

	// Check HTTP status code
	if httpResponse.StatusCode != http.StatusOK {
		// Best-effort error response body read for debugging
		// Intentionally ignoring error - already in error path
		bodyBytes, _ := io.ReadAll(httpResponse.Body) //nolint:errcheck
		return nil, fmt.Errorf("failed to list certificates - [%d] - [%s]",
			httpResponse.StatusCode, string(bodyBytes))
	}

	// Deserialize JSON response (SAME as WorkspacesDB line 96-99)
	listJSON, err := common.DeserializeJSONSnake(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize LIST response: %w", err)
	}

	// Convert to CertificateListResponse struct
	var response CertificateListResponse
	if err := mapstructure.Decode(listJSON, &response); err != nil {
		return nil, fmt.Errorf("failed to decode LIST response: %w", err)
	}

	// Handle nested response structure
	if response.Certificates == nil || response.Certificates.Items == nil {
		// Empty list: No certificates exist (valid state)
		return []CertificateListItem{}, nil
	}

	return response.Certificates.Items, nil
}
