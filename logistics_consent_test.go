package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPublishProofRequiresConsent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/auth/consent/check/u-7/essential" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"granted":false},"error":null,"metadata":{}}`))
	}))
	defer server.Close()
	client := &InfraiClient{BaseURL: server.URL, Key: "test-key", HTTP: server.Client(), Sleep: func(time.Duration) {}}
	decision, err := publishProof(context.Background(), client, ShipmentEvent{ShipmentID: "s-1", UserID: "u-7", Status: "delivered"}, ProofOfDelivery{ShipmentID: "s-1", ObjectName: "pod.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Sent || decision.Reason != "user consent is absent" {
		t.Fatalf("decision = %+v", decision)
	}
}
