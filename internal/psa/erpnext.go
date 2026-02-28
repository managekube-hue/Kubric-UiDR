// Package erpnext provides ERPNext REST API client for billing and ITSM integration
package erpnext

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client is an ERPNext API client
type Client struct {
	BaseURL   string
	APIKey    string
	APISecret string
	HTTPClient *http.Client
}

// NewClient creates a new ERPNext client
func NewClient(baseURL, apiKey, apiSecret string) *Client {
	return &Client{
		BaseURL:   baseURL,
		APIKey:    apiKey,
		APISecret: apiSecret,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Issue represents an ERPNext issue/ticket
type Issue struct {
	Name        string    `json:"name,omitempty"`
	Subject     string    `json:"subject"`
	Description string    `json:"description"`
	Customer    string    `json:"customer"`
	Priority    string    `json:"priority"` // Low, Medium, High, Critical
	IssueType   string    `json:"issue_type"` // Bug, Feature, Question, Incident
	Status      string    `json:"status,omitempty"`
	RaisedBy    string    `json:"raised_by,omitempty"`
	Creation    time.Time `json:"creation,omitempty"`
}

// SalesInvoice represents an ERPNext sales invoice
type SalesInvoice struct {
	Name        string              `json:"name,omitempty"`
	Customer    string              `json:"customer"`
	PostingDate string              `json:"posting_date"`
	Items       []SalesInvoiceItem  `json:"items"`
	Status      string              `json:"status,omitempty"`
}

// SalesInvoiceItem represents a line item
type SalesInvoiceItem struct {
	ItemCode    string  `json:"item_code"`
	ItemName    string  `json:"item_name"`
	Description string  `json:"description,omitempty"`
	Qty         float64 `json:"qty"`
	Rate        float64 `json:"rate"`
	Amount      float64 `json:"amount"`
}

// CreateIssue creates a new issue in ERPNext
func (c *Client) CreateIssue(ctx context.Context, issue *Issue) (*Issue, error) {
	issue.Status = "Open"
	
	data := map[string]interface{}{
		"doctype":     "Issue",
		"subject":     issue.Subject,
		"description": issue.Description,
		"customer":    issue.Customer,
		"priority":    issue.Priority,
		"issue_type":  issue.IssueType,
		"status":      issue.Status,
		"raised_by":   issue.RaisedBy,
	}
	
	var result Issue
	if err := c.doRequest(ctx, "POST", "/api/resource/Issue", data, &result); err != nil {
		return nil, fmt.Errorf("create issue: %w", err)
	}
	
	return &result, nil
}

// GetIssue retrieves an issue by ID
func (c *Client) GetIssue(ctx context.Context, issueID string) (*Issue, error) {
	var result Issue
	if err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/resource/Issue/%s", issueID), nil, &result); err != nil {
		return nil, fmt.Errorf("get issue: %w", err)
	}
	
	return &result, nil
}

// UpdateIssue updates an issue
func (c *Client) UpdateIssue(ctx context.Context, issueID string, updates map[string]interface{}) (*Issue, error) {
	var result Issue
	if err := c.doRequest(ctx, "PUT", fmt.Sprintf("/api/resource/Issue/%s", issueID), updates, &result); err != nil {
		return nil, fmt.Errorf("update issue: %w", err)
	}
	
	return &result, nil
}

// CreateSalesInvoice creates a sales invoice for billing
func (c *Client) CreateSalesInvoice(ctx context.Context, invoice *SalesInvoice) (*SalesInvoice, error) {
	if invoice.PostingDate == "" {
		invoice.PostingDate = time.Now().Format("2006-01-02")
	}
	
	data := map[string]interface{}{
		"doctype":      "Sales Invoice",
		"customer":     invoice.Customer,
		"posting_date": invoice.PostingDate,
		"items":        invoice.Items,
	}
	
	var result SalesInvoice
	if err := c.doRequest(ctx, "POST", "/api/resource/Sales Invoice", data, &result); err != nil {
		return nil, fmt.Errorf("create sales invoice: %w", err)
	}
	
	return &result, nil
}

// doRequest performs an HTTP request to ERPNext API
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	url := c.BaseURL + path
	
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
	}
	
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	
	req.Header.Set("Authorization", fmt.Sprintf("token %s:%s", c.APIKey, c.APISecret))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ERPNext API error: %s", resp.Status)
	}
	
	if result != nil {
		var apiResp struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		
		if err := json.Unmarshal(apiResp.Data, result); err != nil {
			return fmt.Errorf("unmarshal data: %w", err)
		}
	}
	
	return nil
}
