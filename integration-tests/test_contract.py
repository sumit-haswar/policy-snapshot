from __future__ import annotations

import json
import os
import unittest
import urllib.error
import urllib.request
from typing import Any

GATEWAY_URL = os.getenv("GATEWAY_URL", "http://localhost:8080")
DECISION_URL = os.getenv("DECISION_URL", "http://localhost:8081")
REGISTRY_URL = os.getenv("REGISTRY_URL", "http://localhost:8082")


def request_json(
    base_url: str,
    path: str,
    *,
    method: str = "GET",
    payload: dict[str, Any] | None = None,
) -> tuple[int, dict[str, Any], dict[str, str]]:
    body = None if payload is None else json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(
        f"{base_url}{path}",
        data=body,
        method=method,
        headers={"Content-Type": "application/json"} if body is not None else {},
    )
    try:
        with urllib.request.urlopen(request, timeout=3) as response:
            response_body = response.read()
            return response.status, json.loads(response_body), normalized_headers(response.headers)
    except urllib.error.HTTPError as error:
        with error:
            response_body = error.read()
            try:
                parsed = json.loads(response_body) if response_body else {}
            except json.JSONDecodeError:
                parsed = {}
            return error.code, parsed, normalized_headers(error.headers)


def post_bytes(base_url: str, path: str, payload: dict[str, Any]) -> tuple[int, bytes, dict[str, str]]:
    request = urllib.request.Request(
        f"{base_url}{path}",
        data=json.dumps(payload, separators=(",", ":")).encode("utf-8"),
        method="POST",
        headers={"Content-Type": "application/json", "X-Request-ID": "integration-request"},
    )
    try:
        with urllib.request.urlopen(request, timeout=3) as response:
            return response.status, response.read(), normalized_headers(response.headers)
    except urllib.error.HTTPError as error:
        with error:
            return error.code, error.read(), normalized_headers(error.headers)


def normalized_headers(headers: Any) -> dict[str, str]:
    return {name.lower(): value for name, value in headers.items()}


class ServiceContractTests(unittest.TestCase):
    def test_all_services_are_ready(self) -> None:
        for base_url in (GATEWAY_URL, DECISION_URL, REGISTRY_URL):
            status, payload, _ = request_json(base_url, "/health/ready")
            self.assertEqual(status, 200)
            self.assertEqual(payload["status"], "ready")

    def test_bootstrap_decision_contract(self) -> None:
        status, payload, headers = request_json(
            DECISION_URL,
            "/v1/decisions",
            method="POST",
            payload={
                "subject_id": "subject-123",
                "operation": "export",
                "region": "standard",
            },
        )
        self.assertEqual(status, 200)
        self.assertEqual(
            payload,
            {
                "decision": "allow",
                "matched_rule_id": "allow-standard-export",
                "policy_revision": 1,
            },
        )
        self.assertEqual(headers["x-policy-revision"], "1")

    def test_gateway_preserves_authoritative_response(self) -> None:
        request = {
            "subject_id": "subject-456",
            "operation": "read",
            "region": "standard",
        }
        direct_status, direct_body, direct_headers = post_bytes(
            DECISION_URL, "/v1/decisions", request
        )
        gateway_status, gateway_body, gateway_headers = post_bytes(
            GATEWAY_URL, "/v1/decisions", request
        )

        self.assertEqual((direct_status, gateway_status), (200, 200))
        self.assertEqual(gateway_body, direct_body)
        self.assertEqual(
            gateway_headers["x-policy-revision"],
            direct_headers["x-policy-revision"],
        )
        self.assertEqual(gateway_headers["x-request-id"], "integration-request")

    def test_registry_publishes_versioned_pages(self) -> None:
        request_json(
            REGISTRY_URL,
            "/__fixtures/active",
            method="PUT",
            payload={"profile": "valid-v2"},
        )
        manifest_status, manifest, headers = request_json(
            REGISTRY_URL, "/v1/policy-bundles/current"
        )
        page_status, page, _ = request_json(
            REGISTRY_URL, "/v1/policy-bundles/2/pages/0"
        )

        self.assertEqual((manifest_status, page_status), (200, 200))
        self.assertEqual(manifest["revision"], 2)
        self.assertEqual(manifest["page_count"], 2)
        self.assertTrue(manifest["digest"].startswith("sha256:"))
        self.assertEqual(page["revision"], 2)
        self.assertEqual(page["page_index"], 0)
        self.assertEqual(headers["etag"], '"bundle-v2"')

    def test_registry_failure_profile_does_not_affect_serving_path(self) -> None:
        selected_status, _, _ = request_json(
            REGISTRY_URL,
            "/__fixtures/active",
            method="PUT",
            payload={"profile": "unavailable"},
        )
        registry_status, _, _ = request_json(
            REGISTRY_URL, "/v1/policy-bundles/current"
        )
        decision_status, decision, _ = request_json(
            GATEWAY_URL,
            "/v1/decisions",
            method="POST",
            payload={
                "subject_id": "subject-789",
                "operation": "read",
                "region": "standard",
            },
        )

        self.assertEqual(selected_status, 200)
        self.assertEqual(registry_status, 503)
        self.assertEqual(decision_status, 200)
        self.assertEqual(decision["policy_revision"], 1)


if __name__ == "__main__":
    unittest.main()
