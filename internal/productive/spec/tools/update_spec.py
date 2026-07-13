#!/usr/bin/env python3
"""Regenerate internal/productive/spec/*.yaml from Productive's live OpenAPI spec.

Productive publishes one shared `components` section (~400 schemas covering
their entire product) regardless of the `group` query parameter, so this
script downloads the full spec once and trims it down to exactly what
fortyhours calls:

  - productive.yaml    faithful subset (kept paths + transitively referenced
                        components) used as this repo's API reference.
  - models-only.yaml   paths-stripped, schemas-only view of the six resources
                        the CLI reads/writes. This is the only file fed to
                        oapi-codegen: Productive's response/request bodies
                        reference resource properties via deep `$ref`s
                        (`#/components/schemas/resource_booking/properties/x`)
                        that oapi-codegen's operation/response codegen path
                        can't follow (see "unexpected reference depth"), so we
                        only ask it to generate the plain resource models
                        referenced by those attributes and hand-write the
                        generic JSON:API envelope ourselves.

Usage:
    python3 update_spec.py

Requires: requests (or curl fallback), pyyaml.
"""
import copy
import re
import sys
import urllib.request

import yaml

SPEC_URL = "https://developer.productive.io/reference/download_spec"

# (path, methods-to-keep). Only the operations the CLI actually calls are
# kept: full CRUD for time entries/bookings, read-only lookups for
# projects/services/events/people (used to resolve names to ids).
KEEP_OPS = {
    "/api/v2/projects": ["get"],
    "/api/v2/services": ["get"],
    "/api/v2/time_entries": ["get", "post"],
    "/api/v2/time_entries/{id}": ["get", "patch", "delete"],
    "/api/v2/bookings": ["get", "post"],
    "/api/v2/bookings/{id}": ["get", "patch", "delete"],
    "/api/v2/events": ["get"],
    "/api/v2/people": ["get"],
    "/api/v2/timesheets": ["get", "post"],
    "/api/v2/timesheets/{id}": ["get", "delete"],
}

# Schemas fed to oapi-codegen for model generation. Everything else
# (JSON:API relationship/filter helper schemas) is hand-written instead.
MODEL_SCHEMAS = [
    "resource_booking",
    "resource_event",
    "resource_person",
    "resource_project",
    "resource_service",
    "resource_time_entry",
    "resource_timesheet",
    "_meta",
]

# Productive's schemas double as both the resource's actual attribute shape
# and its filter-query parameter shape, and a handful of properties get the
# filter param's scalar type (e.g. a comma-separated string, for
# `?filter[tag_list]=a,b`) even though the resource itself returns an
# array (confirmed against the live API). Drop the incorrect "type" so
# oapi-codegen falls back to `interface{}`, same as it already does for
# untyped fields elsewhere in this spec (e.g. resource_time_entry.approved).
SCALAR_TYPE_OVERRIDES_TO_ANY = {
    ("resource_person", "tag_list"): "comma-separated string in filters; array in the actual resource",
    ("resource_service", "budgets_and_deals"): "boolean in the filter schema; array in the actual resource",
    ("resource_event", "color_id"): "declared integer; the actual resource returns a string",
    ("resource_project", "tag_colors"): "declared string; the actual resource returns an object keyed by tag name",
    ("resource_project", "template"): "declared string; the actual resource returns a boolean",
}


def detype_misdeclared_scalars(schemas):
    for (schema_name, prop_name), _reason in SCALAR_TYPE_OVERRIDES_TO_ANY.items():
        prop = schemas.get(schema_name, {}).get("properties", {}).get(prop_name)
        if prop is not None:
            prop.pop("type", None)

REF_RE = re.compile(r"^#/components/(schemas|parameters|requestBodies|responses|securitySchemes)/([^/]+)")


def find_refs(node, refs):
    if isinstance(node, dict):
        for k, v in node.items():
            if k == "$ref" and isinstance(v, str):
                m = REF_RE.match(v)
                if m:
                    refs.add((m.group(1), m.group(2)))
            else:
                find_refs(v, refs)
    elif isinstance(node, list):
        for item in node:
            find_refs(item, refs)


def fetch_full_spec():
    with urllib.request.urlopen(SPEC_URL) as resp:
        return yaml.safe_load(resp.read())


def build_paths(doc):
    paths = {}
    missing = []
    for p, methods in KEEP_OPS.items():
        if p not in doc["paths"]:
            missing.append(p)
            continue
        kept_methods = {}
        for m in methods:
            if m not in doc["paths"][p]:
                continue
            op = dict(doc["paths"][p][m])
            # Keep only path parameters (e.g. {id}). Drop the filter[]/sort
            # JSON:API query parameters and the header_organization param:
            # the client builds JSON:API filter query strings by hand and
            # sets X-Organization-Id itself, and some of Productive's filter
            # schemas (e.g. filter_person) contain dangling $refs that break
            # strict codegen.
            params = [param for param in op.get("parameters", []) if param.get("in") == "path"]
            if params:
                op["parameters"] = params
            else:
                op.pop("parameters", None)
            kept_methods[m] = op
        paths[p] = kept_methods
    if missing:
        print("WARNING missing paths:", missing, file=sys.stderr)
    return paths


def prune_components(doc, paths):
    components = doc["components"]
    kept = {"schemas": {}, "parameters": {}, "requestBodies": {}, "responses": {}, "securitySchemes": {}}

    frontier = set()
    find_refs(paths, frontier)

    while frontier:
        section, name = frontier.pop()
        if name in kept.get(section, {}):
            continue
        obj = components.get(section, {}).get(name)
        if obj is None:
            continue
        kept.setdefault(section, {})[name] = obj
        new_refs = set()
        find_refs(obj, new_refs)
        for r in new_refs:
            if r[1] not in kept.get(r[0], {}):
                frontier.add(r)

    kept["securitySchemes"] = components.get("securitySchemes", {})
    return kept


def main():
    doc = fetch_full_spec()

    paths = build_paths(doc)
    kept = prune_components(doc, paths)

    reference = {
        "openapi": doc["openapi"],
        "info": doc["info"],
        "servers": doc.get("servers", [{"url": "https://api.productive.io"}]),
        "tags": doc.get("tags", []),
        "paths": paths,
        "components": {k: v for k, v in kept.items() if v},
    }
    with open("productive.yaml", "w") as f:
        yaml.safe_dump(reference, f, sort_keys=False, allow_unicode=True)

    schemas = copy.deepcopy({k: doc["components"]["schemas"][k] for k in MODEL_SCHEMAS if k in doc["components"]["schemas"]})
    detype_misdeclared_scalars(schemas)
    models_only = {
        "openapi": doc["openapi"],
        "info": doc["info"],
        "paths": {},
        "components": {"schemas": schemas},
    }
    with open("models-only.yaml", "w") as f:
        yaml.safe_dump(models_only, f, sort_keys=False, allow_unicode=True)

    print("productive.yaml: paths=%d schemas=%d" % (len(paths), len(kept["schemas"])))
    print("models-only.yaml: schemas=%d" % len(schemas))


if __name__ == "__main__":
    main()
