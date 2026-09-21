from __future__ import annotations

import hashlib
import json
from dataclasses import dataclass

from .models import BundlePage, Manifest, Rule


@dataclass(frozen=True)
class Profile:
    manifest: Manifest
    etag: str
    pages: dict[int, BundlePage]
    manifest_status: int = 200


def canonical_digest(rules: list[Rule]) -> str:
    ordered = sorted(rules, key=lambda rule: (rule.priority, rule.id))
    payload = json.dumps(
        [rule.model_dump() for rule in ordered],
        ensure_ascii=True,
        separators=(",", ":"),
        sort_keys=True,
    ).encode("utf-8")
    return f"sha256:{hashlib.sha256(payload).hexdigest()}"


def profiles() -> dict[str, Profile]:
    valid_rules = [
        Rule(
            effect="deny",
            id="deny-restricted-region",
            operation="*",
            priority=10,
            region="restricted",
        ),
        Rule(
            effect="deny",
            id="deny-standard-export",
            operation="export",
            priority=15,
            region="standard",
        ),
        Rule(
            effect="allow",
            id="allow-read",
            operation="read",
            priority=20,
            region="*",
        ),
        Rule(
            effect="allow",
            id="allow-standard-export",
            operation="export",
            priority=30,
            region="standard",
        ),
    ]
    valid_manifest = Manifest(
        revision=2,
        page_count=2,
        rule_count=len(valid_rules),
        digest=canonical_digest(valid_rules),
    )
    valid_pages = {
        0: BundlePage(revision=2, page_index=0, rules=valid_rules[:2]),
        1: BundlePage(revision=2, page_index=1, rules=valid_rules[2:]),
    }

    return {
        "valid-v2": Profile(valid_manifest, '"bundle-v2"', valid_pages),
        "unavailable": Profile(
            valid_manifest,
            '"bundle-v2-unavailable"',
            valid_pages,
            manifest_status=503,
        ),
    }
