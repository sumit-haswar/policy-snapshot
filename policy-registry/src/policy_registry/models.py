from __future__ import annotations

from pydantic import BaseModel, ConfigDict, Field


class Rule(BaseModel):
    model_config = ConfigDict(extra="forbid")

    effect: str
    id: str
    operation: str
    priority: int
    region: str


class Manifest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    revision: int
    page_count: int
    rule_count: int
    digest: str


class BundlePage(BaseModel):
    model_config = ConfigDict(extra="forbid")

    revision: int
    page_index: int
    rules: list[Rule]


class FixtureSelection(BaseModel):
    model_config = ConfigDict(extra="forbid")

    profile: str = Field(min_length=1)

