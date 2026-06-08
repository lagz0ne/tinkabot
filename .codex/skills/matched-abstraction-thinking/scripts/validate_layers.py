#!/usr/bin/env python3
"""Validate matched-abstraction layer documents."""

from __future__ import annotations

import re
import sys
from pathlib import Path


LAYERS = ("approach", "plan", "task")

REQUIRED_SECTIONS = {
    "approach": ("## Scope", "## Layer Contract", "## Plan-Readiness Gate"),
    "plan": ("## Consumed Approach", "## Decomposition", "## Verification Strategy"),
    "task": ("## Acceptance Contract", "## RED Artifact", "## Verification Evidence"),
}

FORBIDDEN_PATTERNS = {
    "approach": (
        (re.compile(r"(?m)^- \[[ xX]\]"), "approach docs must not contain task checklists"),
        (re.compile(r"(?m)^Run:"), "approach docs must not contain command recipes"),
        (re.compile(r"(?m)^Files:"), "approach docs must not contain file-level work"),
    ),
    "plan": (
        (re.compile(r"(?m)^- \[[ xX]\]"), "plan docs must not contain task checklists"),
        (re.compile(r"(?m)^Files:"), "plan docs must not contain file-level work"),
    ),
}

PLACEHOLDERS = ("TODO", "TBD", "[TODO", "fill in", "implement later")
PROMISSORY_EVIDENCE = ("will be recorded", "recorded later", "to be recorded", "pending")


def parse_frontmatter(path: Path, text: str) -> tuple[dict[str, object], str]:
    if not text.startswith("---\n"):
        raise ValueError(f"{path}: missing frontmatter")

    end = text.find("\n---\n", 4)
    if end == -1:
        raise ValueError(f"{path}: unterminated frontmatter")

    raw = text[4:end].splitlines()
    body = text[end + 5 :]
    data: dict[str, object] = {}
    current_key: str | None = None

    for line in raw:
        stripped = line.strip()
        if not stripped:
            continue
        if stripped.startswith("- "):
            if current_key is None:
                raise ValueError(f"{path}: list item without key in frontmatter")
            data.setdefault(current_key, [])
            value = stripped[2:].strip().strip("\"'")
            if not isinstance(data[current_key], list):
                raise ValueError(f"{path}: mixed scalar/list frontmatter for {current_key}")
            data[current_key].append(value)
            continue
        if ":" not in stripped:
            raise ValueError(f"{path}: invalid frontmatter line: {line}")
        key, value = stripped.split(":", 1)
        key = key.strip()
        value = value.strip()
        current_key = key
        if value == "[]":
            data[key] = []
        elif value:
            data[key] = value.strip("\"'")
            current_key = None
        else:
            data[key] = []

    return data, body


def reference_layer(root: Path, source: Path, ref: str) -> str | None:
    if ref.startswith(("http://", "https://")):
        return None

    target = (source.parent / ref).resolve()
    if not target.exists():
        raise ValueError(f"{source}: reference does not exist: {ref}")

    try:
        rel = target.relative_to(root.resolve())
    except ValueError as exc:
        raise ValueError(f"{source}: reference escapes layer root: {ref}") from exc

    if not rel.parts or rel.parts[0] not in LAYERS:
        raise ValueError(f"{source}: reference is outside known layers: {ref}")
    return rel.parts[0]


def display_path(root: Path, path: Path) -> str:
    try:
        return path.relative_to(root).as_posix()
    except ValueError:
        return str(path)


def section_body(body: str, heading: str) -> str | None:
    pattern = re.compile(rf"(?ms)^{re.escape(heading)}\s*\n(.*?)(?=^##\s|\Z)")
    match = pattern.search(body)
    return match.group(1).strip() if match else None


def has_concrete_verification_evidence(text: str) -> bool:
    lowered = text.lower()
    if not text.strip() or any(phrase in lowered for phrase in PROMISSORY_EVIDENCE):
        return False
    return bool(re.search(r"`[^`]+`\s*->\s*`?[^`\n]+`?", text))


def validate(root: Path) -> list[str]:
    errors: list[str] = []
    seen_layers: set[str] = set()

    if not root.exists():
        return [f"{root}: layer root does not exist"]

    for layer in LAYERS:
        layer_dir = root / layer
        if not layer_dir.is_dir():
            errors.append(f"{layer_dir}: missing layer directory")

    for path in sorted(root.rglob("*.md")):
        rel = path.relative_to(root)
        if len(rel.parts) < 2 or rel.parts[0] not in LAYERS:
            errors.append(f"{rel.as_posix()}: markdown docs must live under approach/, plan/, or task/")
            continue

        text = path.read_text(encoding="utf-8")
        try:
            meta, body = parse_frontmatter(path, text)
        except ValueError as exc:
            errors.append(str(exc))
            continue

        layer = meta.get("layer")
        if layer not in LAYERS:
            errors.append(f"{path}: invalid layer: {layer}")
            continue

        for key in ("topic", "references"):
            if key not in meta:
                errors.append(f"{path}: missing required frontmatter key: {key}")

        if rel.parts[0] != layer:
            errors.append(f"{path}: frontmatter layer does not match directory")
            continue

        seen_layers.add(layer)

        for placeholder in PLACEHOLDERS:
            if placeholder in text:
                errors.append(f"{path}: placeholder text is not allowed: {placeholder}")

        for section in REQUIRED_SECTIONS[layer]:
            if section_body(body, section) is None:
                errors.append(f"{display_path(root, path)}: missing required section: {section}")

        if layer == "task":
            evidence = section_body(body, "## Verification Evidence")
            if evidence is not None and not has_concrete_verification_evidence(evidence):
                errors.append(f"{display_path(root, path)}: verification evidence must contain concrete command/result evidence")

        for pattern, message in FORBIDDEN_PATTERNS.get(layer, ()):
            if pattern.search(body):
                errors.append(f"{path}: {message}")

        refs = meta.get("references", [])
        if not isinstance(refs, list):
            errors.append(f"{path}: references must be a list")
            continue

        for ref in refs:
            try:
                target_layer = reference_layer(root, path, ref)
            except ValueError as exc:
                errors.append(str(exc))
                continue
            if target_layer is None:
                continue
            if layer == "approach" and target_layer != "approach":
                errors.append(f"{path}: approach docs may only reference approach docs")
            if layer == "plan" and target_layer == "task":
                errors.append(f"{path}: plan docs may not reference task docs as authority")

    for layer in LAYERS:
        if layer not in seen_layers:
            errors.append(f"{layer} layer has no markdown docs")

    return errors


def main(argv: list[str]) -> int:
    root = Path(argv[1]) if len(argv) > 1 else Path("docs/matched-abstraction")
    errors = validate(root)
    if errors:
        print("Layer validation failed:")
        for error in errors:
            print(f"- {error}")
        return 1

    print(f"Layer validation passed: {root}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
