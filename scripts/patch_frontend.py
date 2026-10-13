#!/usr/bin/env python3
"""Apply Buildoverse deployment fixes to a compiled SvelteKit frontend."""

from __future__ import annotations

import argparse
import os
import re
import tempfile
from pathlib import Path


FOOTER_LINKS = (
    "let i=[{label:`Terms of Service`,href:`/terms`},"
    "{label:`Privacy Policy`,href:`/privacy`},"
    "{label:`Community Guidelines`,href:`/community-guidelines`},"
    "{label:`Creator Guidelines`,href:`/creator-guidelines`}];"
)
FOOTER_LINKS_WITH_SOURCE = (
    "let i=[{label:`Terms of Service`,href:`/terms`},"
    "{label:`Privacy Policy`,href:`/privacy`},"
    "{label:`Community Guidelines`,href:`/community-guidelines`},"
    "{label:`Creator Guidelines`,href:`/creator-guidelines`},"
    "{label:`Source Code`,href:`https://github.com/iceables/Luduvo-Source-Code`}];"
)
ACCESSORY_RENDER = (
    "for(let e of[...h.failures,...g])console.warn(`[marketplace] Skipped "
    "${e.kind} item ${e.itemId}: ${e.message}`);"
    "if(c||a!==l)return;E(r,{status:`ready`})"
)
ACCESSORY_RENDER_WITH_FALLBACK = (
    "for(let e of[...h.failures,...g])console.warn(`[marketplace] Skipped "
    "${e.kind} item ${e.itemId}: ${e.message}`);"
    "if(c||a!==l)return;"
    "let failures=[...h.failures,...g];"
    "E(r,{status:failures.some(e=>String(e.itemId)===String(t.item.id))?"
    "`error`:`ready`})"
)
BRAND_STYLE = """\n\t\t<style id="buildoverse-brand">
\t\t\ta[aria-label="Buildoverse home"] > svg { display: none !important; }
\t\t\ta[aria-label="Buildoverse home"]::after {
\t\t\t\tcontent: "Buildoverse";
\t\t\t\tdisplay: block;
\t\t\t\tcolor: var(--foreground);
\t\t\t\tfont-size: 1.125rem;
\t\t\t\tfont-weight: 700;
\t\t\t\tline-height: 1.5rem;
\t\t\t\twhite-space: nowrap;
\t\t\t}
\t\t</style>\n"""


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise ValueError(f"Expected one {label} anchor, found {count}")
    return text.replace(old, new, 1)


def atomic_write(path: Path, text: str) -> None:
    with tempfile.NamedTemporaryFile(
        mode="w", encoding="utf-8", dir=path.parent, delete=False
    ) as temporary:
        temporary.write(text)
        temporary_path = Path(temporary.name)
    os.chmod(temporary_path, 0o644)
    os.replace(temporary_path, path)


def patch(dist: Path) -> None:
    index_path = dist / "index.html"
    index = index_path.read_text(encoding="utf-8")
    entry_names = set(
        re.findall(r'/_app/immutable/entry/(app\.[A-Za-z0-9._-]+\.js)', index)
    )
    if len(entry_names) != 1:
        raise ValueError(f"Expected one active SvelteKit app entry, found {entry_names}")
    entry_name = entry_names.pop()
    entry_path = dist / "_app/immutable/entry" / entry_name
    entry = entry_path.read_text(encoding="utf-8")

    dependencies = re.findall(r'["`]\.\./nodes/([^"`]+\.js)["`]', entry)

    def dependency(node_id: str) -> str:
        matches = sorted({name for name in dependencies if name.startswith(node_id + ".")})
        if len(matches) != 1:
            raise ValueError(f"Expected one active node {node_id}, found {matches}")
        return matches[0]

    node0_name = dependency("0")
    node46_name = dependency("46")
    node0_path = dist / "_app/immutable/nodes" / node0_name
    node46_path = dist / "_app/immutable/nodes" / node46_name
    node0 = node0_path.read_text(encoding="utf-8")
    node46 = node46_path.read_text(encoding="utf-8")

    node0 = replace_once(node0, FOOTER_LINKS, FOOTER_LINKS_WITH_SOURCE, "footer link list")
    node0 = replace_once(
        node0,
        'aria-label="Legal"',
        'aria-label="Legal and resources"',
        "footer navigation label",
    )
    node46 = replace_once(
        node46, ACCESSORY_RENDER, ACCESSORY_RENDER_WITH_FALLBACK, "accessory renderer"
    )

    node0_new_name = node0_path.stem + ".buildoverse.js"
    node46_new_name = node46_path.stem + ".buildoverse.js"
    entry_new_name = Path(entry_name).stem + ".buildoverse.js"
    outputs = [
        node0_path.with_name(node0_new_name),
        node46_path.with_name(node46_new_name),
        entry_path.with_name(entry_new_name),
    ]
    if any(path.exists() for path in outputs):
        raise FileExistsError("A versioned Buildoverse frontend asset already exists")

    entry = entry.replace("../nodes/" + node0_name, "../nodes/" + node0_new_name)
    entry = entry.replace("../nodes/" + node46_name, "../nodes/" + node46_new_name)
    index = index.replace(entry_name, entry_new_name)
    index = index.replace(node0_name, node0_new_name)
    if 'id="buildoverse-brand"' in index:
        raise ValueError("Buildoverse brand styles are already installed")
    index = replace_once(index, "</head>", BRAND_STYLE + "\t</head>", "head closing tag")

    for path, text in (
        (outputs[0], node0),
        (outputs[1], node46),
        (outputs[2], entry),
    ):
        atomic_write(path, text)
    atomic_write(index_path, index)
    print(f"Patched {index_path}; accessory fallback and Source Code link are enabled.")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--dist",
        required=True,
        type=Path,
        help="compiled SvelteKit frontend directory containing index.html",
    )
    args = parser.parse_args()
    patch(args.dist.resolve())


if __name__ == "__main__":
    main()
