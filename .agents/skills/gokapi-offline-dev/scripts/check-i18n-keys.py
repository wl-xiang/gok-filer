#!/usr/bin/env python3
"""Validate Gokapi i18n resource files against the template usage.

Checks performed:
  1. Every `{{ tr "key" }}` / `{{ trf "key" ... }}` used in internal/webserver/web/templates/*.tmpl
     exists in the reference locale (default en.json). A missing key silently renders as the
     raw key, so this must be clean.
  2. All locale files define exactly the same key set (no language left behind).
  3. No empty values.

Usage: python check-i18n-keys.py [repo-root]
Exit code 1 if any problem is found.
"""
import glob
import json
import os
import re
import sys

REPO = sys.argv[1] if len(sys.argv) > 1 else "."
LOCALES = os.path.join(REPO, "internal", "i18n", "locales")
TEMPLATES = os.path.join(REPO, "internal", "webserver", "web", "templates", "*.tmpl")
REFERENCE = "en.json"

KEY_PATTERN = re.compile(r'\btrf?\s+"([^"]+)"')


def main():
    files = sorted(glob.glob(os.path.join(LOCALES, "*.json")))
    if not files:
        print("no locale files found in", LOCALES)
        return 1

    locales = {}
    for path in files:
        with open(path, encoding="utf-8") as handle:
            locales[os.path.basename(path)] = json.load(handle)

    if REFERENCE not in locales:
        print("reference locale %s is missing" % REFERENCE)
        return 1
    reference = locales[REFERENCE]

    used = set()
    for path in glob.glob(TEMPLATES):
        with open(path, encoding="utf-8") as handle:
            used.update(KEY_PATTERN.findall(handle.read()))

    problems = 0

    missing = sorted(key for key in used if key not in reference)
    if missing:
        problems += 1
        print("MISSING in %s (%d):" % (REFERENCE, len(missing)))
        for key in missing:
            print("  -", key)

    unused = sorted(set(reference) - used)
    if unused:
        print("INFO: defined but not used in templates (%d): %s" % (len(unused), ", ".join(unused)))

    for name, dictionary in sorted(locales.items()):
        if name == REFERENCE:
            continue
        only_ref = sorted(set(reference) - set(dictionary))
        only_locale = sorted(set(dictionary) - set(reference))
        if only_ref or only_locale:
            problems += 1
            print("%s key mismatch vs %s" % (name, REFERENCE))
            if only_ref:
                print("  missing:", only_ref)
            if only_locale:
                print("  extra:  ", only_locale)

    for name, dictionary in sorted(locales.items()):
        empty = sorted(key for key, value in dictionary.items() if not str(value).strip())
        if empty:
            problems += 1
            print("%s has empty values: %s" % (name, empty))

    print("templates use %d keys; %d locales checked" % (len(used), len(locales)))
    print("RESULT:", "FAIL" if problems else "OK")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
