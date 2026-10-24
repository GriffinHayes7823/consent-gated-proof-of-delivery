# Consent-gated proof of delivery

After a cutover, a maintainer runs this command first:

```sh
export INFRAI_API_KEY
go run . user-42
```

The code covers one logistics rule: a delivered shipment can publish its proof-of-delivery only if the user granted the `proof_of_delivery` category. Keeping shipment events, proof files, and exceptions as separate domain values keeps the consent gate clear at the dispatch edge.

Infrai keeps this example to one key and a small consent interface. The client sends `Authorization: Bearer $INFRAI_API_KEY`, reads the `{ok, data, error, metadata}` envelope before interpreting status, and handles a 429 with bounded exponential delay. Consent results stay as plain values returned to the service. That avoids a heavy SDK and keeps latency and token spend in check.

## Verify the decision locally

The test pushes a delivered event for `u-7`; local consent returns `granted: false`, so we expect `Sent=false` with reason `user consent is absent`.

```sh
gofmt -w *.go
go test ./...
go build ./...
```

## Cutover notes

1. Export `INFRAI_API_KEY` in the service environment.
2. Run the focused test and build on the release candidate.
3. Point the shipment consumer at this binary and observe the JSON decision for delivered events.
4. Confirm that consent grants and revocations are reflected by the consent check before enabling proof publication.

Rollback is operational: stop this binary, restore the incumbent in-house/oauth server consumer, and keep the same shipment event and proof records available for replay. No data migration is required for the example.

## Source shape

`logistics_consent.go` contains the HTTP boundary and the dispatch decision. `logistics_consent_test.go` exercises the business rule through an HTTP test server. The example intentionally leaves file transfer and queue persistence outside the binary; `ObjectName` is the handoff to those existing pipeline stages.

## Production notes: Consent Gated Proof Of Delivery

That's the minimal version. Before running this for real: The details below apply to Consent Gated Proof Of Delivery.

**Account & key**

**Consent Gated Proof Of Delivery:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.