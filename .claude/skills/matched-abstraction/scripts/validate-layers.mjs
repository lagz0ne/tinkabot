#!/usr/bin/env node
import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, isAbsolute, join, relative, resolve } from "node:path";

const layers = ["approach", "plan", "task"];

const required = {
  approach: ["## Scope", "## Layer Contract", "## Plan-Readiness Gate"],
  plan: ["## Consumed Approach", "## Decomposition", "## Verification Strategy"],
  task: ["## Acceptance Contract", "## RED Artifact", "## Verification Evidence"],
};

const forbidden = {
  approach: [
    [/^- \[[ xX]\]/m, "approach docs must not contain task checklists"],
    [/^Run:/m, "approach docs must not contain command recipes"],
    [/^Files:/m, "approach docs must not contain file-level work"],
  ],
  plan: [
    [/^- \[[ xX]\]/m, "plan docs must not contain task checklists"],
    [/^Files:/m, "plan docs must not contain file-level work"],
  ],
};

const placeholders = ["TODO", "TBD", "[TODO", "fill in", "implement later"];
const promissoryEvidence = ["will be recorded", "recorded later", "to be recorded", "pending"];

function esc(text) {
  return text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function mdFiles(root) {
  const out = [];

  function walk(dir) {
    for (const name of readdirSync(dir)) {
      const path = join(dir, name);
      if (statSync(path).isDirectory()) {
        walk(path);
      } else if (path.endsWith(".md")) {
        out.push(path);
      }
    }
  }

  walk(root);
  return out.sort();
}

function display(root, path) {
  const rel = relative(root, path);
  return rel.startsWith("..") || isAbsolute(rel) ? path : rel.replaceAll("\\", "/");
}

function parseFrontmatter(path, text) {
  if (!text.startsWith("---\n")) throw new Error(`${path}: missing frontmatter`);

  const end = text.indexOf("\n---\n", 4);
  if (end === -1) throw new Error(`${path}: unterminated frontmatter`);

  const data = {};
  let current = null;
  for (const line of text.slice(4, end).split("\n")) {
    const s = line.trim();
    if (!s) continue;

    if (s.startsWith("- ")) {
      if (!current) throw new Error(`${path}: list item without key in frontmatter`);
      const list = data[current];
      if (!Array.isArray(list)) throw new Error(`${path}: mixed scalar/list frontmatter for ${current}`);
      list.push(s.slice(2).trim().replace(/^["']|["']$/g, ""));
      continue;
    }

    const colon = s.indexOf(":");
    if (colon === -1) throw new Error(`${path}: invalid frontmatter line: ${line}`);

    const key = s.slice(0, colon).trim();
    const value = s.slice(colon + 1).trim();
    current = key;
    if (value === "[]") {
      data[key] = [];
    } else if (value) {
      data[key] = value.replace(/^["']|["']$/g, "");
      current = null;
    } else {
      data[key] = [];
    }
  }

  return [data, text.slice(end + 5)];
}

function sectionBody(body, heading) {
  const match = new RegExp(`^${esc(heading)}\\s*$`, "m").exec(body);
  if (!match) return null;

  const lineEnd = body.indexOf("\n", match.index);
  const rest = lineEnd === -1 ? "" : body.slice(lineEnd + 1);
  const next = rest.search(/^##\s/m);
  return (next === -1 ? rest : rest.slice(0, next)).trim();
}

function hasConcreteEvidence(text) {
  const lower = text.toLowerCase();
  if (!text.trim() || promissoryEvidence.some((phrase) => lower.includes(phrase))) return false;
  return /`[^`]+`\s*->\s*`?[^`\n]+`?/.test(text);
}

function refLayer(root, source, ref) {
  if (ref.startsWith("http://") || ref.startsWith("https://")) return null;

  const target = resolve(dirname(source), ref);
  if (!existsSync(target)) throw new Error(`${source}: reference does not exist: ${ref}`);

  const rel = relative(resolve(root), target);
  if (rel.startsWith("..") || isAbsolute(rel)) throw new Error(`${source}: reference escapes layer root: ${ref}`);

  const first = rel.split(/[\\/]/)[0];
  if (!layers.includes(first)) throw new Error(`${source}: reference is outside known layers: ${ref}`);
  return first;
}

export function validate(input = "docs/matched-abstraction") {
  const root = resolve(input);
  const errors = [];
  const seen = new Set();

  if (!existsSync(root)) return [`${input}: layer root does not exist`];

  for (const layer of layers) {
    const dir = join(root, layer);
    if (!existsSync(dir) || !statSync(dir).isDirectory()) errors.push(`${dir}: missing layer directory`);
  }

  for (const path of mdFiles(root)) {
    const rel = relative(root, path).split(/[\\/]/);
    if (rel.length < 2 || !layers.includes(rel[0])) {
      errors.push(`${display(root, path)}: markdown docs must live under approach/, plan/, or task/`);
      continue;
    }

    const text = readFileSync(path, "utf8");
    let meta;
    let body;
    try {
      [meta, body] = parseFrontmatter(path, text);
    } catch (err) {
      errors.push(err instanceof Error ? err.message : String(err));
      continue;
    }

    const layer = meta.layer;
    if (!layers.includes(layer)) {
      errors.push(`${path}: invalid layer: ${layer}`);
      continue;
    }

    for (const key of ["topic", "references"]) {
      if (!(key in meta)) errors.push(`${path}: missing required frontmatter key: ${key}`);
    }

    if (rel[0] !== layer) {
      errors.push(`${path}: frontmatter layer does not match directory`);
      continue;
    }

    seen.add(layer);

    for (const placeholder of placeholders) {
      if (text.includes(placeholder)) errors.push(`${path}: placeholder text is not allowed: ${placeholder}`);
    }

    for (const section of required[layer]) {
      if (sectionBody(body, section) === null) errors.push(`${display(root, path)}: missing required section: ${section}`);
    }

    if (layer === "task") {
      const evidence = sectionBody(body, "## Verification Evidence");
      if (evidence !== null && !hasConcreteEvidence(evidence)) {
        errors.push(`${display(root, path)}: verification evidence must contain concrete command/result evidence`);
      }
    }

    for (const [pattern, msg] of forbidden[layer] ?? []) {
      if (pattern.test(body)) errors.push(`${path}: ${msg}`);
    }

    const refs = meta.references ?? [];
    if (!Array.isArray(refs)) {
      errors.push(`${path}: references must be a list`);
      continue;
    }

    for (const ref of refs) {
      try {
        const target = refLayer(root, path, ref);
        if (!target) continue;
        if (layer === "approach" && target !== "approach") errors.push(`${path}: approach docs may only reference approach docs`);
        if (layer === "plan" && target === "task") errors.push(`${path}: plan docs may not reference task docs as authority`);
      } catch (err) {
        errors.push(err instanceof Error ? err.message : String(err));
      }
    }
  }

  for (const layer of layers) {
    if (!seen.has(layer)) errors.push(`${layer} layer has no markdown docs`);
  }

  return errors;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const root = process.argv[2] ?? "docs/matched-abstraction";
  const errors = validate(root);
  if (errors.length) {
    console.error("Layer validation failed:");
    for (const error of errors) console.error(`- ${error}`);
    process.exit(1);
  }

  console.log(`Layer validation passed: ${root}`);
}
