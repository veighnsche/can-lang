"""Build isolated current-Can DI-07 cases with a bundled canlc.

Usage: python3 run.py /absolute/path/to/bundled/canlc
Projects and generated output live under /private/tmp/can-variant-matrix-cases.
"""

import json
import subprocess
import sys
from pathlib import Path


ROOT = Path("/private/tmp/can-variant-matrix-cases")
HEADER = "package app\n    provides []\n    uses [option, codec, bytes]\n"
MAIN = """fn void main
    emits []
    given
        str[] args
    asserts
        empty: [] => ok
    ok
"""
PHANTOM = """record unit
variant tagged<item>
    unit
variant bridge
    unit
"""
BOX = """record box<item>
    item value
"""


def function(result, name, given, assertion, body):
    return (f"fn {result} {name}\n    emits []\n    given\n"
            f"        {given}\n    asserts\n        sample: {assertion}\n"
            f"    {body}\n")


CASES = {
    "phantom-direct": (PHANTOM, function("tagged<str>", "convert", "tagged<int> value", "unit() => ok unit()", "ok value")),
    "phantom-bridge": (PHANTOM, function("tagged<str>", "convert", "tagged<int> value", "unit() => ok unit()", "bridge alias = value\n    ok alias")),
    "phantom-leaf": (PHANTOM, function("tagged<str>", "convert", "tagged<int> value", "unit() => ok unit()", "match value\n        unit => ok value")),
    "option-direct": ("", function("option::value<str>", "convert", "option::value<int> value", "option::none() => ok option::none()", "ok value")),
    "option-none": ("", function("option::value<str>", "convert", "option::value<int> value", "option::none() => ok option::none()", "match value\n        option::none => ok value\n        option::some<int> => ok option::none()")),
    "option-some-int": ("", function("option::value<str>", "convert", "option::some<int> value", "option::some<int>(4) => ok option::none()", "ok value")),
    "option-some-str": ("", function("option::value<str>", "convert", "option::some<str> value", "option::some<str>(\"x\") => ok option::some<str>(\"x\")", "ok value")),
    "option-none-leaf": ("", function("option::value<str>", "convert", "option::none value", "option::none() => ok option::none()", "ok value")),
    "record-direct": (BOX, function("box<str>", "convert", "box<int> value", "box<int>(4) => ok box<str>(\"4\")", "ok value")),
    "record-same": (BOX, function("box<int>", "convert", "box<int> value", "box<int>(4) => ok box<int>(4)", "ok value")),
    "codec-tagged": (PHANTOM, """fn tagged<int> roundtrip
    emits [codec::invalid_data]
    given
        tagged<int> value
    asserts
        sample: unit() => ok unit()
    match chain
        call codec::encode_json<tagged<int>>(value) as bytes::buffer encoded
        call codec::decode_json<tagged<int>>(encoded) as tagged<int> decoded
        codec::invalid_data
        ok => ok decoded
"""),
    "codec-tagged-cross": (PHANTOM, """fn tagged<str> cross_decode
    emits [codec::invalid_data]
    given
        tagged<int> value
    asserts
        sample: unit() => ok unit()
    match chain
        call codec::encode_json<tagged<int>>(value) as bytes::buffer encoded
        call codec::decode_json<tagged<str>>(encoded) as tagged<str> decoded
        codec::invalid_data
        ok => ok decoded
"""),
    "codec-option-int-none": ("", """fn option::value<int> roundtrip
    emits [codec::invalid_data]
    given
        option::value<int> value
    asserts
        sample: option::none() => ok option::none()
    match chain
        call codec::encode_json<option::value<int>>(value) as bytes::buffer encoded
        call codec::decode_json<option::value<int>>(encoded) as option::value<int> decoded
        codec::invalid_data
        ok => ok decoded
"""),
    "codec-option-none-cross": ("", """fn option::value<str> cross_decode
    emits [codec::invalid_data]
    given
        option::value<int> value
    asserts
        absent: option::none() => ok option::none()
    match chain
        call codec::encode_json<option::value<int>>(value) as bytes::buffer encoded
        call codec::decode_json<option::value<str>>(encoded) as option::value<str> decoded
        codec::invalid_data
        ok => ok decoded
"""),
    "codec-option-str-some": ("", """fn option::value<str> roundtrip
    emits [codec::invalid_data]
    given
        option::value<str> value
    asserts
        sample: option::some<str>("x") => ok option::some<str>("x")
    match chain
        call codec::encode_json<option::value<str>>(value) as bytes::buffer encoded
        call codec::decode_json<option::value<str>>(encoded) as option::value<str> decoded
        codec::invalid_data
        ok => ok decoded
"""),
    "codec-option-cross-tag": ("", """fn option::value<str> decode_wrong_leaf
    emits [codec::invalid_data]
    given
        str text
    asserts
        wrong_specialization: "{\\\"case\\\":\\\"option::some<int>\\\",\\\"value\\\":{\\\"value\\\":4}}" => codec::invalid_data("/case", "variant_tag")
    match chain
        call bytes::from_utf8(text) as bytes::buffer encoded
        call codec::decode_json<option::value<str>>(encoded) as option::value<str> decoded
        codec::invalid_data
        ok => ok decoded
"""),
}
REJECTED = {"phantom-direct", "option-direct", "option-some-int", "record-direct"}


def main():
    compiler = Path(sys.argv[1]).resolve()
    ROOT.mkdir(exist_ok=True)
    failures = []
    for name, (declarations, body) in CASES.items():
        project = ROOT / name
        source = project / "src" / "main.can"
        source.parent.mkdir(parents=True, exist_ok=True)
        source.write_text(HEADER + declarations + body + MAIN)
        (project / "can.project.json").write_text(json.dumps({"source_root": "src", "error_registry": "can.errors.json"}))
        (project / "can.errors.json").write_text('{"active":[],"retired":[]}')
        run = subprocess.run([str(compiler), "build", str(project)], capture_output=True, text=True)
        output = (run.stderr + "\n" + run.stdout).strip().splitlines()
        result = {"case": name, "exit": run.returncode}
        if run.returncode == 0:
            report = json.loads(output[-1])
            result["validation"] = report["validation"]
            result["assertions"] = report["assertions"]
        else:
            result["diagnostic"] = output[-1]
        print(json.dumps(result))
        if name in REJECTED:
            if run.returncode == 0 or "expression type does not fit expected type" not in result["diagnostic"]:
                failures.append(name)
        elif run.returncode != 0 or result["validation"] != "verified" or result["assertions"]["failed"] != 0:
            failures.append(name)
    if failures:
        raise SystemExit("unexpected outcomes: " + ", ".join(failures))


if __name__ == "__main__":
    main()
