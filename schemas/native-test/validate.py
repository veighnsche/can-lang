#!/usr/bin/env python3
"""Validate native-test wire documents against schemas/native-test/ schemas.

Implements a documented JSON Schema subset over the standard library only:
type, properties, required, additionalProperties (bool), items, minItems,
maxItems, maxProperties, minimum, maximum, exclusiveMinimum,
exclusiveMaximum, minLength, maxLength, pattern, enum, const, anyOf, oneOf,
allOf, $ref (local "#/$defs/..." only). Metadata keywords ($schema, $id,
title, description, $comment) are ignored. Any other keyword fails loudly
so schemas cannot silently use unsupported features.

Beyond shape, enforces the README semantic rules (operation kind/outcome
agreement, limit ordering, sealed completeness, report matched/blocked
conditions, clean-receipt emptiness).

Usage: python3 schemas/native-test/validate.py
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


def semantic(schema_name, doc):
    """Cross-field rules from README; returns a list of error strings."""
    if schema_name == "operation":
        outcome, kind = doc.get("outcome"), doc.get("kind")
        if outcome is not None:
            if outcome == "completed" and kind != "ok":
                return ["outcome completed requires kind ok"]
            if outcome != "completed" and kind == "ok":
                return [f"outcome {outcome} requires a non-ok kind"]
        return []
    if schema_name == "limit":
        budgets = doc.get("budgets", {})
        problems = []
        try:
            if budgets["finalCleanupReserveMs"] > budgets["runWallMs"]:
                problems.append("finalCleanupReserveMs exceeds runWallMs")
            if budgets["prepareMs"] > budgets["runWallMs"]:
                problems.append("prepareMs exceeds runWallMs")
        except KeyError:
            pass
        return problems
    if schema_name == "completeness":
        problems = []
        total = doc.get("totalCount")
        returned = doc.get("returnedCount")
        if isinstance(total, int) and isinstance(returned, int) and returned > total:
            problems.append("returnedCount exceeds totalCount")
        if doc.get("state") == "sealed":
            if doc.get("truncated") is not False:
                problems.append("sealed completeness must not be truncated")
            if doc.get("gaps"):
                problems.append("sealed completeness must have no gaps")
            if total != returned:
                problems.append("sealed completeness requires returnedCount == totalCount")
        return problems
    if schema_name == "report":
        problems = []
        units = doc.get("units", [])
        if doc.get("mode") == "execute" and not units:
            problems.append("execute mode requires at least one unit")
        for unit in units:
            where = unit.get("caseId", "?")
            if unit.get("behavior") == "matched":
                if unit.get("execution") != "completed":
                    problems.append(f"{where}: matched behavior requires completed execution")
                if unit.get("bodyTerminal") is not True:
                    problems.append(f"{where}: matched behavior requires bodyTerminal")
                for check in unit.get("checks", []):
                    if check.get("disposition") != "matched":
                        problems.append(f"{where}: matched behavior requires every check matched")
                        break
            if unit.get("admission") == "blocked":
                if unit.get("execution") != "not-started":
                    problems.append(f"{where}: blocked admission requires not-started execution")
                if unit.get("behavior") != "undetermined":
                    problems.append(f"{where}: blocked admission requires undetermined behavior")
        return problems
    if schema_name == "receipt":
        if doc.get("cleanup") == "clean" and doc.get("remaining"):
            return ["clean cleanup requires empty remaining"]
        return []
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
        schema_name = name if name in schemas else None
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
