#!/usr/bin/env python3
"""Validate bilingual documentation pairs and repository-local Markdown links."""

from __future__ import annotations

import re
import sys
import unicodedata
from pathlib import Path
from urllib.parse import unquote


ROOT = Path(__file__).resolve().parents[1]

PAIRS = {
    "README.md": "README.es.md",
    "CONTRIBUTING.md": "CONTRIBUTING.es.md",
    "SECURITY.md": "SECURITY.es.md",
    "SUPPORT.md": "SUPPORT.es.md",
    "CODE_OF_CONDUCT.md": "CODE_OF_CONDUCT.es.md",
    "docs/CONFIGURATION.md": "docs/es/CONFIGURATION.md",
    "docs/INTEGRATIONS.md": "docs/es/INTEGRATIONS.md",
}

FORBIDDEN_PUBLIC_DOCS = (
    "docs/ARCHITECTURE.md",
    "docs/ROADMAP.md",
    "docs/es/ARCHITECTURE.md",
    "docs/es/ROADMAP.md",
)

LINK_RE = re.compile(r"(?<!!)\[[^\]]*\]\(([^)\s]+)(?:\s+['\"][^'\"]*['\"])?\)")
HREF_RE = re.compile(r"href=['\"]([^'\"]+)['\"]")
EXPLICIT_ANCHOR_RE = re.compile(r"<a\s+(?:name|id)=['\"]([^'\"]+)['\"]", re.IGNORECASE)
FENCE_RE = re.compile(r"^```(yaml|bash|text)\s*$\n(.*?)^```\s*$", re.MULTILINE | re.DOTALL)


def github_slug(value: str) -> str:
    value = re.sub(r"<[^>]+>", "", value)
    value = re.sub(r"[`*_~]", "", value).strip().lower()
    value = "".join(ch for ch in value if not unicodedata.category(ch).startswith(("P", "S")) or ch == "-")
    return re.sub(r"\s+", "-", value)


def anchors(text: str) -> set[str]:
    found = set(EXPLICIT_ANCHOR_RE.findall(text))
    counts: dict[str, int] = {}
    for line in text.splitlines():
        match = re.match(r"^#{1,6}\s+(.+?)\s*#*\s*$", line)
        if not match:
            continue
        base = github_slug(match.group(1))
        index = counts.get(base, 0)
        counts[base] = index + 1
        found.add(base if index == 0 else f"{base}-{index}")
    return found


def local_links(text: str) -> set[str]:
    return set(LINK_RE.findall(text)) | set(HREF_RE.findall(text))


def technical_blocks(text: str) -> list[tuple[str, str]]:
    return [(language, body.rstrip()) for language, body in FENCE_RE.findall(text)]


def main() -> int:
    errors: list[str] = []

    for relative in FORBIDDEN_PUBLIC_DOCS:
        if (ROOT / relative).exists():
            errors.append(f"private planning document present in public tree: {relative}")

    for english_name, spanish_name in PAIRS.items():
        english = ROOT / english_name
        spanish = ROOT / spanish_name
        if not english.is_file():
            errors.append(f"missing canonical document: {english_name}")
            continue
        if not spanish.is_file():
            errors.append(f"missing Spanish translation: {spanish_name}")
            continue

        english_text = english.read_text(encoding="utf-8")
        spanish_text = spanish.read_text(encoding="utf-8")
        if "canonical-language: en" not in english_text:
            errors.append(f"missing canonical-language marker: {english_name}")
        if "translation-source:" not in spanish_text:
            errors.append(f"missing translation-source marker: {spanish_name}")
        if spanish.name not in english_text:
            errors.append(f"English document does not link to Spanish: {english_name}")
        if english.name not in spanish_text:
            errors.append(f"Spanish document does not link to English: {spanish_name}")
        if technical_blocks(english_text) != technical_blocks(spanish_text):
            errors.append(f"technical code blocks differ between {english_name} and {spanish_name}")

    markdown_files = sorted(ROOT.glob("*.md")) + sorted((ROOT / "docs").rglob("*.md"))
    anchor_cache: dict[Path, set[str]] = {}
    for document in markdown_files:
        text = document.read_text(encoding="utf-8")
        for raw_target in local_links(text):
            if raw_target.startswith(("http://", "https://", "mailto:", "oai-library://")):
                continue
            path_part, separator, fragment = raw_target.partition("#")
            target = document if not path_part else (document.parent / unquote(path_part)).resolve()
            try:
                target.relative_to(ROOT)
            except ValueError:
                errors.append(f"link escapes repository: {document.relative_to(ROOT)} -> {raw_target}")
                continue
            if not target.exists():
                errors.append(f"broken local link: {document.relative_to(ROOT)} -> {raw_target}")
                continue
            if separator and fragment and target.suffix.lower() == ".md":
                if target not in anchor_cache:
                    anchor_cache[target] = anchors(target.read_text(encoding="utf-8"))
                if unquote(fragment) not in anchor_cache[target]:
                    errors.append(f"missing anchor: {document.relative_to(ROOT)} -> {raw_target}")

    for relative in ("docs/INTEGRATIONS.md", "docs/es/INTEGRATIONS.md"):
        text = (ROOT / relative).read_text(encoding="utf-8")
        matrix = re.search(r"^## (?:Compatibility matrix|Matriz de compatibilidad)$(.*?)(?=^## )", text, re.MULTILINE | re.DOTALL)
        if not matrix:
            errors.append(f"compatibility matrix not found: {relative}")
            continue
        product_links = re.findall(r"\]\(#([^)]+)\)", matrix.group(1))
        explicit = set(EXPLICIT_ANCHOR_RE.findall(text))
        if len(product_links) != 30:
            errors.append(f"expected 30 product links in {relative}, found {len(product_links)}")
        missing = sorted(set(product_links) - explicit)
        if missing:
            errors.append(f"missing product anchors in {relative}: {', '.join(missing)}")

    if errors:
        print("Documentation validation failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print(f"Documentation OK: {len(PAIRS)} bilingual pairs, local links valid, 30 integration anchors per language.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
