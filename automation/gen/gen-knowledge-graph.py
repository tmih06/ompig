#!/usr/bin/env python3
"""Render docs/knowledge-graph/ompig.graph.json into its published forms.

Outputs: the agent docs page (internal/pigdocs/content/knowledge-graph.md), the
site page body (docs/site/docs/knowledge-graph.md), JSON-LD for tools
(docs/knowledge-graph/ompig-knowledge-graph.jsonld, which the site serves at
/ompig-knowledge-graph.jsonld), and Mermaid source
(docs/knowledge-graph/ompig.mmd). It also copies docs/site/docs/install-troubleshooting.md
into the agent bundle as troubleshooting.md. --check fails when any output is stale.
"""
import json, pathlib, re, sys

root = pathlib.Path(__file__).resolve().parents[2]
graph = json.loads((root / "docs/knowledge-graph/ompig.graph.json").read_text())
entities = {e["id"]: e for e in graph["entities"]}
for source, _, target, _ in graph["relations"]:
    if source not in entities or target not in entities:
        sys.exit(f"relation references an unknown entity: {source} -> {target}")

def tables(doc_link):
    lines = ["| Entity | What it is | Where it lives | Inspect with | Read |", "|---|---|---|---|---|"]
    for e in graph["entities"]:
        lines.append(f"| {e['label']} | {e['summary']} | `{e['where']}` | `{e['inspect']}` | {doc_link(e['doc'])} |")
    lines += ["", "| From | Relation | To | Note |", "|---|---|---|---|"]
    for source, relation, target, note in graph["relations"]:
        lines.append(f"| {entities[source]['label']} | {relation.replace('_', ' ')} | {entities[target]['label']} | {note} |")
    return "\n".join(lines) + "\n"

# Crow's-foot cardinality per relation; unlisted relations are one-to-many.
cardinality = {"implements": "||--||", "is_a": "}o--||", "owns": "||--||", "pins": "||--||", "selects": "}o--o{",
               "written_with": "}o--||", "realized_as": "}o--||", "realizes": "}o--||", "drives": "}o--||",
               "streams_through": "}o--||", "documents_difference_from": "}o--||"}
mermaid = ["erDiagram"]
for source, relation, target, _ in graph["relations"]:
    mermaid.append(f'    {source.upper()} {cardinality.get(relation, "||--o{")} {target.upper()} : "{relation.replace("_", " ")}"')
mermaid_text = "\n".join(mermaid) + "\n"

jsonld = {
    "@context": {"@vocab": "https://schema.org/", "ompig": "https://github.com/tmih06/ompig/terms#", "relations": "ompig:relation"},
    "@graph": [{"@id": f"ompig:{e['id']}", "@type": "DefinedTerm", "name": e["label"], "description": e["summary"], "ompig:kind": e["kind"], "ompig:where": e["where"], "ompig:inspect": e["inspect"]} for e in graph["entities"]]
    + [{"@type": "ompig:Relation", "ompig:from": f"ompig:{s}", "ompig:relation": r, "ompig:to": f"ompig:{t}", "description": n} for s, r, t, n in graph["relations"]],
}

agent = ("# Knowledge graph\n\nThe entities Ompig is built from, how they relate, where each lives, and the command that inspects it. "
         "Use it to locate a concept before reading its page; each row links the page to read next.\n\n"
         + tables(lambda d: f"[{d}]({d})")
         + "\nThe machine-readable form is `ompig-knowledge-graph.jsonld` on the documentation site.\n")
site = ("# Knowledge graph\n\nOmpig is built from a small set of entities. This page lists each entity, where it lives, the command that inspects it, and how the entities relate. "
        "The same graph is published as JSON-LD at [ompig-knowledge-graph.jsonld](/ompig-knowledge-graph.jsonld) and as Mermaid source in `docs/knowledge-graph/ompig.mmd`, "
        "and it ships inside every ompig binary as `knowledge-graph.md` in the local docs (`ompig docs show knowledge-graph`).\n\n"
        "![Entity relationship diagram of Ompig](/ompig-knowledge-graph.svg)\n\n"
        + tables(lambda d: f"`{d}`")
        + "\nThe graph source is `docs/knowledge-graph/ompig.graph.json`. Run `automation/gen/gen-knowledge-graph.py` after editing it.\n")

troubleshooting = (root / "docs/site/docs/install-troubleshooting.md").read_text()
site_url = "https://ompig.dev/docs/latest/"
bundle = root / "internal/pigdocs/content"
# Link to bundled pages locally; pages that exist only on the site keep a site URL.
troubleshooting = re.sub(r"\]\(/docs/latest/([a-z0-9-]+)\)",
                         lambda m: f"]({m[1]}.md)" if (bundle / f"{m[1]}.md").exists() else f"]({site_url}{m[1]}/)",
                         troubleshooting)
outputs = {
    root / "internal/pigdocs/content/troubleshooting.md": troubleshooting,
    root / "internal/pigdocs/content/knowledge-graph.md": agent,
    root / "docs/site/docs/knowledge-graph.md": site,
    root / "docs/knowledge-graph/ompig-knowledge-graph.jsonld": json.dumps(jsonld, indent=2) + "\n",
    root / "docs/knowledge-graph/ompig.mmd": mermaid_text,
}
stale = [str(p.relative_to(root)) for p, text in outputs.items() if not p.exists() or p.read_text() != text]
if "--check" in sys.argv:
    if stale:
        sys.exit("stale knowledge-graph outputs; run: make generate (or: make knowledge-graph), then commit the result: " + ", ".join(stale))
    sys.exit(0)
for path, text in outputs.items():
    path.write_text(text)
print("wrote", ", ".join(str(p.relative_to(root)) for p in outputs))
