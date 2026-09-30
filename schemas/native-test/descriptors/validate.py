#!/usr/bin/env python3
"""Validate K17 descriptor/environment wire documents.

Mirrors the P01 schemas/native-test/validate.py pattern: a documented JSON
Schema subset over the standard library only (type, properties, required,
additionalProperties (bool), items, minItems, maxItems, maxProperties,
minimum, maximum, exclusiveMinimum, exclusiveMaximum, minLength, maxLength,
pattern, enum, const, anyOf, oneOf, allOf, $ref local "#/$defs/..." only).
Metadata keywords ($schema, $id, title, description, $comment) are ignored.
Any other keyword fails loudly so schemas cannot silently use unsupported
features.

Beyond shape, enforces the README semantic rules (fixed fd 0/1/2/3/4
directions, hold-lease byte rules, byte/EOF/lifetime coherence, fresh
snapshot binding, c-cli/direct-entry argv agreement, canonical env names).

Usage: python3 schemas/native-test/descriptors/validate.py
Exit 0 when every valid fixture passes and every invalid fixture fails
with its manifest's expected error substring.
"""
import json
import re
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
META_KEYWORDS = {"$schema", "$id", "title", "description", "$comment", "$defs"}
SUPPORTED_KEYWORDS = META_KEYWORDS | {
    "type", "properties", "required", "additionalProperties", "items",
    "minItems", "maxItems", "maxProperties", "minimum", "maximum",
    "exclusiveMinimum", "exclusiveMaximum", "minLength", "maxLength",
    "pattern", "enum", "const", "anyOf", "oneOf", "allOf", "$ref",
}


class SchemaError(ValueError):
    pass


def check_keywords(schema, path="$"):
    if not isinstance(schema, dict):
        raise SchemaError(f"{path}: schema node must be an object")
    for key, value in schema.items():
        if key not in SUPPORTED_KEYWORDS:
            raise SchemaError(f"{path}: unsupported keyword {key!r}")
    for key in ("properties", "$defs"):
        if isinstance(schema.get(key), dict):
            for name, sub in schema[key].items():
                check_keywords(sub, f"{path}.{key}.{name}")
    for key in ("items",):
        if key in schema:
            check_keywords(schema[key], f"{path}.{key}")
    for key in ("anyOf", "oneOf", "allOf"):
        for i, sub in enumerate(schema.get(key, [])):
            check_keywords(sub, f"{path}.{key}[{i}]")
    if "$ref" in schema and (
        not isinstance(schema["$ref"], str) or not schema["$ref"].startswith("#/$defs/")
    ):
        raise SchemaError(f"{path}: only local '#/$defs/...' refs supported")


def resolve(root, ref):
    node = root
    for part in ref.lstrip("#/").split("/"):
        node = node[part]
    return node


def type_ok(value, name):
    if name == "object":
        return isinstance(value, dict)
    if name == "array":
        return isinstance(value, list)
    if name == "string":
        return isinstance(value, str)
    if name == "boolean":
        return isinstance(value, bool)
    if name == "integer":
        return isinstance(value, int) and not isinstance(value, bool)
    if name == "number":
        return isinstance(value, (int, float)) and not isinstance(value, bool)
    if name == "null":
        return value is None
    raise SchemaError(f"unknown type {name!r}")


def validate(root, schema, value, path="$", errors=None):
    if errors is None:
        errors = []
    if "$ref" in schema:
        validate(root, resolve(root, schema["$ref"]), value, path, errors)
        return errors
    expected = schema.get("type")
    if expected is not None:
        names = expected if isinstance(expected, list) else [expected]
        if not any(type_ok(value, name) for name in names):
            errors.append(f"{path}: expected type {expected}, got {type(value).__name__}")
            return errors
    if "const" in schema and value != schema["const"]:
        errors.append(f"{path}: expected const {schema['const']!r}, got {value!r}")
    if "enum" in schema and value not in schema["enum"]:
        errors.append(f"{path}: {value!r} is not one of {schema['enum']!r}")
    if isinstance(value, str):
        if "minLength" in schema and len(value) < schema["minLength"]:
            errors.append(f"{path}: shorter than minLength {schema['minLength']}")
        if "maxLength" in schema and len(value) > schema["maxLength"]:
            errors.append(f"{path}: longer than maxLength {schema['maxLength']}")
        if "pattern" in schema and not re.search(schema["pattern"], value):
            errors.append(f"{path}: {value!r} does not match {schema['pattern']!r}")
    if isinstance(value, (int, float)) and not isinstance(value, bool):
        for key, ok, msg in (
            ("minimum", value >= schema.get("minimum", value), "below minimum"),
            ("maximum", value <= schema.get("maximum", value), "above maximum"),
            ("exclusiveMinimum", value > schema.get("exclusiveMinimum", value - 1), "at or below exclusiveMinimum"),
            ("exclusiveMaximum", value < schema.get("exclusiveMaximum", value + 1), "at or above exclusiveMaximum"),
        ):
            if key in schema and not ok:
                errors.append(f"{path}: {msg} {schema[key]}")
    if isinstance(value, dict):
        for name in schema.get("required", []):
            if name not in value:
                errors.append(f"{path}: missing required property {name!r}")
        props = schema.get("properties", {})
        for name, item in value.items():
            if name in props:
                validate(root, props[name], item, f"{path}.{name}", errors)
            elif schema.get("additionalProperties") is False:
                errors.append(f"{path}: unknown property {name!r}")
        if "maxProperties" in schema and len(value) > schema["maxProperties"]:
            errors.append(f"{path}: more than maxProperties {schema['maxProperties']}")
    if isinstance(value, list):
        if "minItems" in schema and len(value) < schema["minItems"]:
            errors.append(f"{path}: fewer than minItems {schema['minItems']}")
        if "maxItems" in schema and len(value) > schema["maxItems"]:
            errors.append(f"{path}: more than maxItems {schema['maxItems']}")
        if "items" in schema:
            for i, item in enumerate(value):
                validate(root, schema["items"], item, f"{path}[{i}]", errors)
    for key in ("allOf",):
        for sub in schema.get(key, []):
            validate(root, sub, value, path, errors)
    if "anyOf" in schema:
        branch_errors = []
        for sub in schema["anyOf"]:
            trial = validate(root, sub, value, path, [])
            if not trial:
                break
            branch_errors.append(trial)
        else:
            errors.append(f"{path}: matches no anyOf branch: {branch_errors!r}")
    if "oneOf" in schema:
        passed = sum(1 for sub in schema["oneOf"] if not validate(root, sub, value, path, []))
        if passed != 1:
            errors.append(f"{path}: expected exactly one oneOf match, got {passed}")
    return errors


# Fixed K17 direction contract: fd -> direction from the child side.
FD_DIRECTIONS = {0: "in", 1: "out", 2: "out", 3: "in", 4: "hold"}
SEALED_EOF = {"eof-sent", "eof-observed", "closed"}


def semantic_descriptor_map(doc):
    problems = []
    entries = doc.get("descriptors", [])
    fds = [entry.get("fd") for entry in entries if isinstance(entry, dict)]
    if sorted(fd for fd in fds if isinstance(fd, int)) != [0, 1, 2, 3, 4]:
        problems.append("descriptors must list fds 0,1,2,3,4 exactly once")
    for entry in entries:
        if not isinstance(entry, dict):
            continue
        fd = entry.get("fd")
        where = f"fd {fd}"
        if fd in FD_DIRECTIONS and entry.get("direction") != FD_DIRECTIONS[fd]:
            problems.append(f"{where} requires direction {FD_DIRECTIONS[fd]!r}")
        offered = entry.get("bytes_offered")
        accepted = entry.get("bytes_accepted")
        if isinstance(offered, int) and isinstance(accepted, int) and accepted > offered:
            problems.append(f"{where}: bytes_accepted exceeds bytes_offered")
        byte_state = entry.get("byte_state")
        eof_state = entry.get("eof_state")
        if fd == 4:
            if byte_state != "none" or eof_state != "none":
                problems.append(f"{where}: hold lease carries no byte/eof state")
            if offered not in (0, None) or accepted not in (0, None):
                if offered != 0 or accepted != 0:
                    problems.append(f"{where}: hold lease carries no bytes")
        elif fd in FD_DIRECTIONS:
            if byte_state == "none" or eof_state == "none":
                problems.append(f"{where} requires tracked byte/eof state")
            if byte_state == "exhausted" and eof_state not in SEALED_EOF:
                problems.append(f"{where}: exhausted bytes require sealed eof")
            if eof_state == "closed" and byte_state != "exhausted":
                problems.append(f"{where}: closed eof requires exhausted bytes")
        if entry.get("lifetime") == "released" and eof_state not in ("closed", "none"):
            problems.append(f"{where}: released lifetime requires closed eof")
    return problems


def semantic_environment(doc):
    problems = []
    entry = doc.get("entry")
    argv = doc.get("argv", [])
    if entry == "c-cli" and not argv:
        problems.append("c-cli entry requires argv with argv0")
    if entry == "direct-entry" and argv:
        problems.append("direct-entry entry requires empty argv")
    names = doc.get("env_names", [])
    if isinstance(names, list) and all(isinstance(item, str) for item in names):
        if len(set(names)) != len(names):
            problems.append("duplicate env name")
        if list(names) != sorted(names):
            problems.append("env names must be sorted")
    return problems


def semantic(schema_name, doc):
    """Cross-field rules from README; returns a list of error strings."""
    if schema_name == "descriptor-map":
        return semantic_descriptor_map(doc)
    if schema_name == "environment":
        return semantic_environment(doc)
    return []


def load_schemas():
    schemas = {}
    for path in sorted(HERE.glob("*.schema.json")):
        schemas[path.stem.replace(".schema", "")] = json.loads(path.read_text())
    for name, schema in schemas.items():
        check_keywords(schema, f"{name}.schema.json")
    return schemas


def main():
    schemas = load_schemas()
    failures = []
    valid_dir = HERE / "fixtures" / "valid"
    valid_files = sorted(valid_dir.glob("*.json"))
    for path in valid_files:
        name = path.stem.rsplit("-", 1)[0] if "-" in path.stem else path.stem
        # Multi-word schema names keep every segment but the last fixture tag.
        schema_name = None
        for candidate in sorted(schemas, key=len, reverse=True):
            if path.stem == candidate or path.stem.startswith(candidate + "-"):
                schema_name = candidate
                break
        if schema_name is None:
            failures.append(f"{path.name}: no schema for prefix {name!r}")
            continue
        doc = json.loads(path.read_text())
        problems = validate(schemas[schema_name], schemas[schema_name], doc) + semantic(schema_name, doc)
        if problems:
            failures.append(f"{path.name}: valid fixture rejected: {problems[0]}")
        else:
            print(f"ok valid/{path.name}")
    manifest = json.loads((HERE / "fixtures" / "invalid" / "manifest.json").read_text())
    for filename, rule in sorted(manifest.items()):
        path = HERE / "fixtures" / "invalid" / filename
        doc = json.loads(path.read_text())
        schema_name = rule["schema"]
        problems = validate(schemas[schema_name], schemas[schema_name], doc) + semantic(schema_name, doc)
        if not problems:
            failures.append(f"{filename}: invalid fixture accepted, expected {rule['expect']} ")
        elif rule["expect"] not in " ".join(problems):
            failures.append(f"{filename}: rejection {problems[0]!r} lacks {rule['expect']!r}")
        else:
            print(f"ok invalid/{filename} ({rule['expect']})")
    print(f"{len(valid_files)} valid, {len(manifest)} invalid fixtures checked")
    if failures:
        for failure in failures:
            print("FAIL:", failure)
        return 1
    print("all schema fixtures pass")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
