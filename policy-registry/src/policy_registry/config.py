from __future__ import annotations

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Settings:
    host: str = "127.0.0.1"
    port: int = 8082
    initial_profile: str = "valid-v2"
    enable_fixture_admin: bool = False

    @classmethod
    def from_environment(cls) -> "Settings":
        return cls(
            host=os.getenv("REGISTRY_HOST", "127.0.0.1"),
            port=_positive_int("REGISTRY_PORT", 8082),
            initial_profile=os.getenv("REGISTRY_PROFILE", "valid-v2"),
            enable_fixture_admin=_boolean("REGISTRY_ENABLE_FIXTURE_ADMIN", False),
        )


def _positive_int(name: str, fallback: int) -> int:
    raw = os.getenv(name)
    if raw is None:
        return fallback
    value = int(raw)
    if value <= 0:
        raise ValueError(f"{name} must be positive")
    return value


def _boolean(name: str, fallback: bool) -> bool:
    raw = os.getenv(name)
    if raw is None:
        return fallback
    normalized = raw.lower()
    if normalized in {"1", "true", "yes"}:
        return True
    if normalized in {"0", "false", "no"}:
        return False
    raise ValueError(f"{name} must be a boolean")
