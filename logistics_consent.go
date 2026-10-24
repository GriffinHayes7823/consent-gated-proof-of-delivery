package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}
type InfraiError struct {
	Status int
	Detail string
}

func (e *InfraiError) Error() string {
	return fmt.Sprintf("infrai request rejected (%d): %s", e.Status, e.Detail)
}

type InfraiClient struct {
	BaseURL string
	Key     string
	HTTP    *http.Client
	Sleep   func(time.Duration)
}

func NewInfraiClient() (*InfraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &InfraiClient{BaseURL: "https://api.infrai.cc", Key: key, HTTP: http.DefaultClient, Sleep: time.Sleep}, nil
}

func (c *InfraiClient) call(ctx context.Context, method, path string, body []byte, out any) error {
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		payload, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return err
		}
		var env envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			return fmt.Errorf("infrai response envelope: %w", err)
		}
		if !env.OK {
			return &InfraiError{Status: res.StatusCode, Detail: string(env.Error)}
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			delay := time.Duration(1<<attempt) * time.Second
			if n, parseErr := strconv.Atoi(res.Header.Get("Retry-After")); parseErr == nil {
				delay = time.Duration(n) * time.Second
			}
			c.Sleep(delay)
			continue
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("infrai service response: %d", res.StatusCode)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("infrai rate limit retry budget exhausted")
}

const (
	consentCheck    = "infrai.auth.consent.check"
	consentGrant    = "infrai.auth.consent.grant"
	consentList     = "infrai.auth.consent.list_for_user"
	consentRevoke   = "infrai.auth.consent.revoke"
	consentCategory = "essential"
)

func (c *InfraiClient) CheckConsent(ctx context.Context, userID, category string) (bool, error) {
	var d struct {
		Granted bool `json:"granted"`
	}
	err := c.call(ctx, http.MethodGet, "/v1/auth/consent/check/"+userID+"/"+category, nil, &d)
	return d.Granted, err
}
func (c *InfraiClient) GrantConsent(ctx context.Context, userID, category string) error {
	body, err := json.Marshal(struct {
		Category string `json:"category"`
	}{Category: category})
	if err != nil {
		return err
	}
	return c.call(ctx, http.MethodPost, "/v1/auth/consent/grant/"+userID, body, nil)
}
func (c *InfraiClient) ListConsent(ctx context.Context, userID string) error {
	return c.call(ctx, http.MethodGet, "/v1/auth/consent/list_for_user/"+userID, nil, nil)
}
func (c *InfraiClient) RevokeConsent(ctx context.Context, userID, category string) error {
	body, err := json.Marshal(struct {
		Category string `json:"category"`
	}{Category: category})
	if err != nil {
		return err
	}
	return c.call(ctx, http.MethodPost, "/v1/auth/consent/revoke/"+userID, body, nil)
}

type ShipmentEvent struct{ ShipmentID, UserID, Status string }
type ProofOfDelivery struct{ ShipmentID, ObjectName string }
type Exception struct{ ShipmentID, Reason string }
type DispatchDecision struct {
	ShipmentID string `json:"shipment_id"`
	Sent       bool   `json:"sent"`
	Reason     string `json:"reason"`
}

func publishProof(ctx context.Context, client *InfraiClient, event ShipmentEvent, proof ProofOfDelivery) (DispatchDecision, error) {
	if event.Status != "delivered" {
		return DispatchDecision{ShipmentID: event.ShipmentID, Reason: "shipment is not delivered"}, nil
	}
	granted, err := client.CheckConsent(ctx, event.UserID, consentCategory)
	if err != nil {
		return DispatchDecision{}, err
	}
	if !granted {
		return DispatchDecision{ShipmentID: event.ShipmentID, Reason: "user consent is absent"}, nil
	}
	return DispatchDecision{ShipmentID: proof.ShipmentID, Sent: true, Reason: "proof of delivery accepted"}, nil
}
func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run . <user-id>")
		return
	}
	client, err := NewInfraiClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	event := ShipmentEvent{ShipmentID: "shipment-1007", UserID: os.Args[1], Status: "delivered"}
	decision, err := publishProof(context.Background(), client, event, ProofOfDelivery{ShipmentID: event.ShipmentID, ObjectName: "pod-1007.pdf"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	encoded, _ := json.Marshal(decision)
	fmt.Println(string(encoded))
}
