package loops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client implements the Loops API for a given API Key / endpoint.
type Client struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

// DefaultEndpoint is the default endpoint used for the Loops API.
const DefaultEndpoint = "https://app.loops.so/api/v1"

// NewClient creates a new Client object.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:   apiKey,
		endpoint: DefaultEndpoint,
		client:   http.DefaultClient,
	}
}

// WithEndpoint attaches a non-default endpoint to the Client.
// This is generally used with dedicated, or non-serverless deployments.
func (c *Client) WithEndpoint(endpoint string) *Client {
	// Ensure that the endpoint doesn't end with a trailing slash.
	c.endpoint = strings.TrimSuffix(endpoint, "/")
	return c
}

// WithHTTPClient attaches a non-default HTTP client to the Client.
func (c *Client) WithHTTPClient(client *http.Client) *Client {
	c.client = client
	return c
}

func (c *Client) doRequest(ctx context.Context, method, path string, body, dst any) error {
	var reqBody io.Reader
	if body != nil {
		marshalled, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(marshalled)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, reqBody)
	if err != nil {
		return err
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	client := c.client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		buf, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, buf)
	}

	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			return err
		}
	}

	return nil
}

type CreateContactResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
}

func validateFields(
	ctx context.Context,
	fields map[string]interface{},
) (map[string]interface{}, error) {
	ret := map[string]interface{}{}
	for k, v := range fields {
		// Don't allow the email field to be set.
		if k == "email" {
			continue
		}
		switch v.(type) {
		case string, bool, int:
		default:
			return nil,fmt.Errorf("invalid field type for %s: %T", k, v)
		}
		ret[k] = v
	}
	return ret, nil
}

// CreateContact creates a new contact in Loops.
func (c *Client) CreateContact(
	ctx context.Context,
	email string,
	// Fields is a map of field names to values. Values can only be string, boolean or int
	fields map[string]interface{},
	) (*CreateContactResponse, error) {
	req, err := validateFields(ctx, fields)
	if err != nil {
		return nil, err
	}
	req["email"] = email
	var resp CreateContactResponse
	if err := c.doRequest(ctx, "POST", "/contacts/create", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

type UpsertContactResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
}

// UpsertContact updates or creates a contact in Loops.
func (c *Client) UpsertContact(ctx context.Context, email string, fields map[string]interface{}) (*UpsertContactResponse, error) {
	req, err := validateFields(ctx, fields)
	if err != nil {
		return nil, err
	}
	req["email"] = email
	var resp UpsertContactResponse
	if err := c.doRequest(ctx, "PUT", "/contacts/update", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

type DeleteContactResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// DeleteContact deletes a contact from Loops.
func (c *Client) DeleteContact(ctx context.Context, email string) (*DeleteContactResponse, error) {
	req := map[string]interface{}{
		"email": email,
	}
	var resp DeleteContactResponse
	if err := c.doRequest(ctx, "DELETE", "/contacts/delete", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

type SendEventResponse struct {
	Success bool `json:"success"`
}

type SendEventRequest struct {
	Email     string `json:"email"`
	EventName string `json:"eventName"`
}

// Send Event sends an event Loop to a contact.
// WARNING: This will create a contact if it doesn't exist.
func (c *Client) SendEvent(ctx context.Context, req SendEventRequest) (*SendEventResponse, error) {
	var resp SendEventResponse
	if err := c.doRequest(ctx, "POST", "/events/send", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
