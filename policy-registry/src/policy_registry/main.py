from __future__ import annotations

import logging
import threading
import time
import uuid
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

import uvicorn
from fastapi import FastAPI, Header, Request
from fastapi.responses import JSONResponse, Response

from .config import Settings
from .fixtures import Profile, profiles
from .models import FixtureSelection

LOGGER = logging.getLogger("policy_registry.http")


class FixtureState:
    def __init__(self, initial: str, available: dict[str, Profile]) -> None:
        if initial not in available:
            raise ValueError(f"unknown initial fixture profile {initial!r}")
        self._available = available
        self._active = initial
        self._lock = threading.Lock()

    def active(self) -> tuple[str, Profile]:
        with self._lock:
            return self._active, self._available[self._active]

    def select(self, name: str) -> bool:
        with self._lock:
            if name not in self._available:
                return False
            self._active = name
            return True

    def names(self) -> list[str]:
        return sorted(self._available)


def create_app(settings: Settings | None = None) -> FastAPI:
    settings = settings or Settings.from_environment()
    state = FixtureState(settings.initial_profile, profiles())

    @asynccontextmanager
    async def lifespan(_: FastAPI) -> AsyncIterator[None]:
        yield

    app = FastAPI(title="Synthetic Policy Registry", version="1.0.0", lifespan=lifespan)
    app.state.fixtures = state

    @app.middleware("http")
    async def observe(request: Request, call_next):  # type: ignore[no-untyped-def]
        started = time.perf_counter()
        request_id = request.headers.get("x-request-id") or str(uuid.uuid4())
        response = await call_next(request)
        response.headers["x-request-id"] = request_id
        LOGGER.info(
            "request completed",
            extra={
                "method": request.method,
                "route": route_template(request.url.path),
                "status": response.status_code,
                "duration_ms": round((time.perf_counter() - started) * 1000, 2),
            },
        )
        return response

    @app.get("/health/live")
    def live() -> dict[str, str]:
        return {"status": "ok"}

    @app.get("/health/ready")
    def ready() -> dict[str, str]:
        return {"status": "ready"}

    @app.get("/v1/policy-bundles/current")
    async def current_manifest(if_none_match: str | None = Header(default=None)):
        _, profile = state.active()
        if profile.manifest_status != 200:
            return error_response(
                profile.manifest_status,
                "REGISTRY_UNAVAILABLE",
                "policy bundle is temporarily unavailable",
            )
        if if_none_match == profile.etag:
            return Response(status_code=304, headers={"ETag": profile.etag})
        return JSONResponse(
            status_code=200,
            content=profile.manifest.model_dump(),
            headers={"ETag": profile.etag},
        )

    @app.get("/v1/policy-bundles/{revision}/pages/{page_index}")
    def bundle_page(revision: int, page_index: int):
        _, profile = state.active()
        if revision != profile.manifest.revision:
            return error_response(404, "NOT_FOUND", "bundle revision not found")
        page = profile.pages.get(page_index)
        if page is None:
            return error_response(404, "NOT_FOUND", "bundle page not found")
        return page.model_dump()

    if settings.enable_fixture_admin:

        @app.get("/__fixtures/active")
        def active_fixture() -> dict[str, object]:
            active, _ = state.active()
            return {"profile": active, "available": state.names()}

        @app.put("/__fixtures/active")
        def select_fixture(selection: FixtureSelection):
            if not state.select(selection.profile):
                return error_response(400, "UNKNOWN_PROFILE", "fixture profile does not exist")
            return {"profile": selection.profile}

    return app


def error_response(status: int, code: str, message: str) -> JSONResponse:
    return JSONResponse(
        status_code=status,
        content={"error": {"code": code, "message": message}},
    )


def route_template(path: str) -> str:
    if path.startswith("/v1/policy-bundles/") and "/pages/" in path:
        return "/v1/policy-bundles/{revision}/pages/{page_index}"
    return path


def run() -> None:
    settings = Settings.from_environment()
    uvicorn.run(create_app(settings), host=settings.host, port=settings.port, access_log=False)


if __name__ == "__main__":
    run()
