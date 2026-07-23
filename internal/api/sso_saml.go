package api

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/securelens/securelens/internal/events"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

// ssoSAMLInitiate starts the SAML login flow
func (s *Server) ssoSAMLInitiate(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "provider_id")
	tenantSlug := r.URL.Query().Get("tenant")

	if tenantSlug == "" {
		pkg.Error(w, fmt.Errorf("tenant parameter required"), http.StatusBadRequest)
		return
	}

	var tenantID string
	err := s.db.GetContext(r.Context(), &tenantID,
		"SELECT id FROM tenants WHERE slug = $1", tenantSlug)
	if err != nil {
		pkg.Error(w, fmt.Errorf("tenant not found"), http.StatusNotFound)
		return
	}

	var provider store.SSOProvider
	err = s.db.GetContext(r.Context(), &provider,
		`SELECT id, tenant_id, name, type, enabled, idp_entity_id, idp_sso_url, 
		        idp_certificate, sp_entity_id
		 FROM sso_providers WHERE id = $1 AND tenant_id = $2 AND enabled = true AND type = 'saml'`,
		providerID, tenantID)
	if err != nil {
		pkg.Error(w, fmt.Errorf("SSO provider not found or disabled"), http.StatusNotFound)
		return
	}

	if provider.IDPSSOURL == nil || provider.IDPEntityID == nil {
		pkg.Error(w, fmt.Errorf("SAML provider not properly configured"), http.StatusBadRequest)
		return
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://app.securelens.ai"
	}

	// Generate relay state
	relayState := generateState()

	// Store session
	_, err = s.db.ExecContext(r.Context(),
		`INSERT INTO sso_sessions (tenant_id, provider_id, state, expires_at)
		 VALUES ($1, $2, $3, NOW() + INTERVAL '10 minutes')`,
		tenantID, provider.ID, relayState)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	// Build SAML AuthnRequest
	acsURL := fmt.Sprintf("%s/api/v1/auth/sso/saml/acs", baseURL)
	spEntityID := fmt.Sprintf("%s/api/v1/auth/sso/saml/metadata/%s", baseURL, tenantID)
	if provider.SPEntityID != nil && *provider.SPEntityID != "" {
		spEntityID = *provider.SPEntityID
	}

	authnRequest := buildSAMLAuthnRequest(spEntityID, acsURL, *provider.IDPSSOURL)

	// Encode and redirect
	encodedRequest := base64.StdEncoding.EncodeToString([]byte(authnRequest))
	redirectURL := fmt.Sprintf("%s?SAMLRequest=%s&RelayState=%s",
		*provider.IDPSSOURL,
		url.QueryEscape(encodedRequest),
		url.QueryEscape(relayState))

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// ssoSAMLACS handles the SAML Assertion Consumer Service callback
func (s *Server) ssoSAMLACS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		redirectToLoginWithError(w, r, "Invalid SAML response")
		return
	}

	samlResponse := r.FormValue("SAMLResponse")
	relayState := r.FormValue("RelayState")

	if samlResponse == "" {
		redirectToLoginWithError(w, r, "No SAML response received")
		return
	}

	// Look up session by relay state
	var session struct {
		TenantID   string `db:"tenant_id"`
		ProviderID string `db:"provider_id"`
	}
	err := s.db.GetContext(ctx, &session,
		`SELECT tenant_id, provider_id FROM sso_sessions 
		 WHERE state = $1 AND expires_at > NOW()`, relayState)
	if err != nil {
		redirectToLoginWithError(w, r, "SSO session expired or invalid")
		return
	}

	// Delete the session
	s.db.ExecContext(ctx, "DELETE FROM sso_sessions WHERE state = $1", relayState)

	// Get provider config
	var provider store.SSOProvider
	err = s.db.GetContext(ctx, &provider,
		`SELECT id, tenant_id, name, idp_certificate, attribute_mapping, 
		        default_role, auto_create_users
		 FROM sso_providers WHERE id = $1`, session.ProviderID)
	if err != nil {
		redirectToLoginWithError(w, r, "SSO provider not found")
		return
	}

	// Decode SAML response
	responseXML, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		redirectToLoginWithError(w, r, "Invalid SAML response encoding")
		return
	}

	// Parse SAML response using custom struct
	var response SAMLResponse
	if err := xml.Unmarshal(responseXML, &response); err != nil {
		log.Error().Err(err).Msg("Failed to parse SAML response")
		redirectToLoginWithError(w, r, "Invalid SAML response format")
		return
	}

	// Basic validation
	if response.Status.StatusCode.Value != "urn:oasis:names:tc:SAML:2.0:status:Success" {
		redirectToLoginWithError(w, r, "SAML authentication failed")
		return
	}

	if response.Assertion == nil {
		redirectToLoginWithError(w, r, "No assertion in SAML response")
		return
	}

	assertion := response.Assertion

	// Extract user attributes
	var email, name, subjectID string
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		subjectID = assertion.Subject.NameID.Value
	}

	for _, attr := range assertion.AttributeStatement.Attributes {
		switch attr.Name {
		case "email", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress":
			if len(attr.Values) > 0 {
				email = attr.Values[0].Value
			}
		case "name", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
			"http://schemas.microsoft.com/identity/claims/displayname":
			if len(attr.Values) > 0 {
				name = attr.Values[0].Value
			}
		}
	}

	if email == "" {
		// Try to use NameID as email if it looks like an email
		if subjectID != "" && len(subjectID) > 3 && subjectID[0] != '_' {
			email = subjectID
		} else {
			redirectToLoginWithError(w, r, "Email not provided by identity provider")
			return
		}
	}

	// Find or create user
	user, err := s.findOrCreateSSOUser(ctx, session.TenantID, provider, subjectID, email, name)
	if err != nil {
		log.Error().Err(err).Msg("Failed to find/create SSO user")
		redirectToLoginWithError(w, r, "Failed to create user account")
		return
	}

	// Generate JWT
	token, err := pkg.GenerateToken(user.ID, user.TenantID, []string{}, user.IsSuperAdmin)
	if err != nil {
		redirectToLoginWithError(w, r, "Failed to generate session")
		return
	}

	// Update last login
	s.db.ExecContext(ctx, "UPDATE users SET last_login_at = $1 WHERE id = $2", time.Now(), user.ID)

	// Audit log
	clientIP := pkg.GetClientIP(r)
	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   user.TenantID,
		UserID:     user.ID,
		Action:     "user.sso_login",
		Resource:   "user",
		ResourceID: user.ID,
		Details:    store.JSON(fmt.Sprintf(`{"provider":"%s","email":"%s","type":"saml"}`, provider.Name, user.Email)),
		IP:         clientIP,
	})

	events.Emit("user.sso_login", map[string]string{
		"user_id":     user.ID,
		"provider_id": provider.ID,
		"email":       user.Email,
		"type":        "saml",
	})

	// Redirect to frontend with token
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "https://app.securelens.ai"
	}
	redirectURL := fmt.Sprintf("%s/auth/sso-callback?token=%s", frontendURL, token)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// ssoSAMLMetadata returns the SP metadata for SAML configuration
func (s *Server) ssoSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://app.securelens.ai"
	}

	entityID := fmt.Sprintf("%s/api/v1/auth/sso/saml/metadata/%s", baseURL, tenantID)
	acsURL := fmt.Sprintf("%s/api/v1/auth/sso/saml/acs", baseURL)

	metadata := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="%s">
  <md:SPSSODescriptor AuthnRequestsSigned="false" WantAssertionsSigned="true" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat>
    <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="%s" index="0" isDefault="true"/>
  </md:SPSSODescriptor>
</md:EntityDescriptor>`, entityID, acsURL)

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(metadata))
}

func buildSAMLAuthnRequest(spEntityID, acsURL, destination string) string {
	id := "_" + generateState()
	issueInstant := time.Now().UTC().Format(time.RFC3339)

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="%s"
    Version="2.0"
    IssueInstant="%s"
    Destination="%s"
    AssertionConsumerServiceURL="%s"
    ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST">
  <saml:Issuer>%s</saml:Issuer>
  <samlp:NameIDPolicy Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress" AllowCreate="true"/>
</samlp:AuthnRequest>`, id, issueInstant, destination, acsURL, spEntityID)
}

// SAML Response XML structures for parsing
type SAMLResponse struct {
	XMLName   xml.Name        `xml:"Response"`
	Status    SAMLStatus      `xml:"Status"`
	Assertion *SAMLAssertion  `xml:"Assertion"`
}

type SAMLStatus struct {
	StatusCode SAMLStatusCode `xml:"StatusCode"`
}

type SAMLStatusCode struct {
	Value string `xml:"Value,attr"`
}

type SAMLAssertion struct {
	Subject            *SAMLSubject           `xml:"Subject"`
	AttributeStatement SAMLAttributeStatement `xml:"AttributeStatement"`
}

type SAMLSubject struct {
	NameID *SAMLNameID `xml:"NameID"`
}

type SAMLNameID struct {
	Value string `xml:",chardata"`
}

type SAMLAttributeStatement struct {
	Attributes []SAMLAttribute `xml:"Attribute"`
}

type SAMLAttribute struct {
	Name   string           `xml:"Name,attr"`
	Values []SAMLAttrValue  `xml:"AttributeValue"`
}

type SAMLAttrValue struct {
	Value string `xml:",chardata"`
}
