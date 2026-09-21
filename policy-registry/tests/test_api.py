from __future__ import annotations

from fastapi.testclient import TestClient

from policy_registry.config import Settings
from policy_registry.main import create_app


def test_valid_manifest_pages_and_conditional_request() -> None:
    app = create_app(Settings(enable_fixture_admin=True))
    with TestClient(app) as client:
        manifest = client.get("/v1/policy-bundles/current")
        page_zero = client.get("/v1/policy-bundles/2/pages/0")
        unchanged = client.get(
            "/v1/policy-bundles/current",
            headers={"If-None-Match": manifest.headers["etag"]},
        )

    assert manifest.status_code == 200
    assert manifest.json()["revision"] == 2
    assert manifest.json()["page_count"] == 2
    assert manifest.json()["digest"] == (
        "sha256:00812c5276238b2410a813dc0de75d6c4b829d1ee1542332c1491a913ef61ecc"
    )
    assert page_zero.status_code == 200
    assert page_zero.json()["page_index"] == 0
    assert unchanged.status_code == 304


def test_fixture_admin_is_absent_by_default() -> None:
    app = create_app(Settings(enable_fixture_admin=False))
    with TestClient(app) as client:
        response = client.get("/__fixtures/active")
    assert response.status_code == 404


def test_unknown_fixture_is_rejected() -> None:
    app = create_app(Settings(enable_fixture_admin=True))
    with TestClient(app) as client:
        response = client.put("/__fixtures/active", json={"profile": "not-real"})
    assert response.status_code == 400
    assert response.json()["error"]["code"] == "UNKNOWN_PROFILE"
