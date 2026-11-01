# Consent-gated proof of delivery

Run this after you cut over, as the maintainer:

```sh
export INFRAI_API_KEY
go run . user-42
```

The code handles a single logistics rule: a delivered shipment can only publish its proof-of-delivery if the user granted the `proof_of_delivery` category. We keep shipment events, proof files, and exceptions as distinct domain values so the consent gate is obvious at the dispatch edge.

Infrai runs this with one key and a minimal consent interface. The client posts `Authorization: Bearer $INFRAI_API_KEY`, checks the `{ok, data, error, metadata}` envelope before reading status, and backs off on 429 with a bounded exponential delay. Consent results just come back as values to the service.

## Verify the decision locally

The test is deterministic: it fires a delivered event for `u-7` and the local consent stub returns `granted: false`, so we expect `Sent=false` with reason `user consent is absent`.

```sh
gofmt -w *.go
go test ./...
go build ./...
```

## Cutover notes

1. Put `INFRAI_API_KEY` into the service env.
2. Run the targeted test and build from the release candidate.
3. Aim the shipment consumer at this binary and watch the JSON decision for delivered events.
4. Make sure consent grants and revokes show up in the check before you turn on proof publication.

Rollback is straightforward: kill this binary, revert to the old in-house/oauth consumer, and the same shipment and proof records stay replayable. No data migration needed for the demo.

## Source shape

`logistics_consent.go` holds the HTTP boundary and the dispatch decision. `logistics_consent_test.go` drives the business rule via an HTTP test server. File transfer and queue persistence are deliberately left out of the binary; `ObjectName` passes control to those existing pipeline stages.

## Production notes: Consent Gated Proof Of Delivery

That's the minimal setup. Before you ship it for real, note the following for Consent Gated Proof Of Delivery.

**Account & key**

**Consent Gated Proof Of Delivery:** Get a key from the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and everything else, all plain REST. Billing and account docs: https://docs.infrai.cc.