"""Bounded research instrument, not Can syntax or a production HTML checker.

Only this page's pure names/fields/string literals and already prepared opaque
attributes/nodes are supported. Existing Can checking validates their types;
the existing HTML runtime still validates dynamic splices. No raw HTML path.
"""
from dataclasses import dataclass, field
from pathlib import Path
import re


class TreeError(ValueError):
    def __init__(self, code, line, message, related=None):
        self.diagnostic = dict(code=code, line=line, message=message, related=related)
        super().__init__(f"{line}: {code}: {message}")


@dataclass
class Entry:
    kind: str
    value: str
    line: int
    attrs: list[str] = field(default_factory=list)
    children: list = field(default_factory=list)
    column: int = 1


def pure_value(value, line, allow_string=False):
    if re.fullmatch(r"[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)*", value):
        return value
    # Research subset shared by Can and JSON: do not assign JSON semantics
    # to Can's NUL or unknown escapes. Forward admitted source verbatim.
    if allow_string and re.fullmatch(r'"(?:[^"\\\x00-\x1f]|\\["\\nrt])*"', value):
        return value
    raise TreeError("HTML-TREE-PREPARE", line, "prepare calls and complex expressions in ordinary Can before this tree")


def parse(text):
    # Reuse the current runtime inventory as evidence; never silently invent tags.
    runtime = Path(__file__).resolve().parents[4] / "runtime/platform/html.ts"
    policy = re.search(r'export const authorTags = new Set\(\s*"([^"]+)"', runtime.read_text())
    if policy is None:
        raise RuntimeError("runtime tag inventory changed; re-scope this research instrument")
    tags = set(policy.group(1).split())
    rows = [(i, s) for i, s in enumerate(text.splitlines(), 1) if s.strip()]
    if not rows or not rows[0][1].startswith("tree document "):
        raise TreeError("HTML-TREE-DOCUMENT", 1, "expected tree document <title>")
    first_line, first_row = rows[0]
    title = pure_value(first_row[14:].strip(), first_line, True)
    root = Entry("document", title, first_line, column=15)
    stack = [(0, root)]
    for number, row in rows[1:]:
        width = len(row) - len(row.lstrip(" "))
        if "\t" in row or width == 0 or width % 4:
            raise TreeError("HTML-TREE-INDENT", number, "expected four-space tree indentation")
        depth, content = width // 4, row.strip()
        while stack[-1][0] >= depth:
            stack.pop()
        parent = stack[-1][1]
        if depth != stack[-1][0] + 1:
            raise TreeError("HTML-TREE-INDENT", number, "indentation skips a parent")
        if content in ("head", "body"):
            entry = Entry(content, "", number)
            if parent is not root or any(c.kind == content for c in root.children):
                raise TreeError("HTML-TREE-SECTION", number, "head/body appear once directly under document")
        elif content.startswith("element "):
            match = re.fullmatch(r"element ([a-z][a-z0-9]*) \[(.*)\]", content)
            if not match or match[1] not in tags:
                raise TreeError("HTML-TREE-TAG", number, "expected an admitted literal HTML tag and attribute list")
            slots = match[2].split(",") if match[2].strip() else []
            if any(not slot.strip() for slot in slots):
                raise TreeError("HTML-TREE-ATTRS", number, "attribute lists cannot contain empty slots")
            attrs = [pure_value(s.strip(), number) for s in slots]
            if len(set(attrs)) != len(attrs):
                raise TreeError("HTML-TREE-DUPLICATE", number, "the same immutable attribute appears twice")
            entry = Entry("element", match[1], number, attrs)
            if entry.value == "form":
                for _, ancestor in stack:
                    if ancestor.kind == "element" and ancestor.value == "form":
                        raise TreeError("HTML-TREE-NESTED-FORM", number, "literal form is inside another form", ancestor.line)
        else:
            kind, _, value = content.partition(" ")
            if kind not in ("text", "node", "nodes"):
                raise TreeError("HTML-TREE-ENTRY", number, "expected element, text, node or nodes")
            entry = Entry(kind, pure_value(value, number, kind == "text"), number)
        if entry.value:
            entry.column = width + content.index(entry.value) + 1
        if parent.kind not in ("document", "head", "body", "element"):
            raise TreeError("HTML-TREE-LEAF", number, "a splice or text row cannot contain children", parent.line)
        if parent.kind == "document" and entry.kind not in ("head", "body"):
            raise TreeError("HTML-TREE-SECTION", number, "document children must be head and body")
        if parent.kind == "head" and entry.kind != "node" and entry.kind != "nodes":
            raise TreeError("HTML-TREE-HEAD", number, "this prototype's head accepts prepared head nodes only")
        if parent.kind == "element" and parent.value in ("input", "br", "hr", "img"):
            raise TreeError("HTML-TREE-VOID", number, "void element cannot have children", parent.line)
        parent.children.append(entry)
        stack.append((depth, entry))
    if [c.kind for c in root.children] != ["head", "body"]:
        raise TreeError("HTML-TREE-SECTION", 1, "expected head followed by body")
    return root


def lower(root):
    lines, mapping = [], []
    serial = 0

    def emit(text, entry, embedded=()):
        lines.append("        " + text)
        # Columns are 1-based source byte/scalar columns for this ASCII probe.
        # Record embedded splice spans too: their errors belong to the splice
        # row, even when Can checks it inside its parent's generated call.
        expressions = []
        for expression, source, start in embedded:
            expressions.append(dict(generated_start=9 + start, generated_end=9 + start + len(expression), tree_line=source.line, tree_column=source.column, tree_expression=source.value))
        mapping.append(dict(generated_line=len(lines), tree_line=entry.line, operation=entry.kind, expressions=expressions))

    def children(entry):
        result, refs = "[", []
        for child in entry.children:
            if len(result) > 1:
                result += ", "
            value = walk(child)
            if child.kind in ("node", "nodes"):
                # The generated spread punctuation is not part of the authored
                # expression; map the name itself so columns remain aligned.
                refs.append((child.value, child, len(result) + (3 if child.kind == "nodes" else 0)))
            result += value
        return result + "]", refs

    def walk(entry):
        nonlocal serial
        if entry.kind == "node":
            return entry.value
        if entry.kind == "nodes":
            return "..." + entry.value
        serial += 1
        name = f"tree_node_{serial}"
        if entry.kind == "text":
            emit(f"call html::text({entry.value}) as html::node {name}", entry, [(entry.value, entry, len("call html::text("))])
        elif entry.kind == "element":
            kids, refs = children(entry)
            emit(f'call html::make_tag("{entry.value}") as html::tag {name}_tag', entry)
            prefix = f"call html::element({name}_tag, [{', '.join(entry.attrs)}], "
            emit(f"{prefix}{kids}) as html::node {name}", entry, [(v, e, len(prefix) + i) for v, e, i in refs])
        return name

    (head, head_refs), (body, body_refs) = (children(section) for section in root.children)
    prefix = f"call html::document({root.value}, "
    refs = [(v, e, len(prefix) + i) for v, e, i in head_refs]
    refs += [(v, e, len(prefix) + len(head) + 2 + i) for v, e, i in body_refs]
    emit(f"{prefix}{head}, {body}) as html::safe document", root, refs)
    return "\n".join(lines) + "\n", mapping
