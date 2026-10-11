"""Read-only source inventory. Never emits matched credential values or uploads files."""
import csv
import hashlib
import json
import os
import re
from collections import Counter, defaultdict
from pathlib import Path

WORKSPACE = Path(__file__).resolve().parent.parent
SOURCE = WORKSPACE / "hw-bridge-handoff-20260929"
OUT = Path(__file__).resolve().parent
SKIP_DIRS = {".git", ".gradle", "build", "node_modules", "vendor", ".idea", "runtime", "__pycache__"}
TEXT_EXT = {".go", ".java", ".kt", ".php", ".py", ".ps1", ".sh", ".json", ".yml", ".yaml", ".xml", ".gradle", ".properties", ".md", ".txt", ".sql", ".js", ".html"}
ARCHIVE_EXT = {".apk", ".aab", ".zip", ".gz", ".7z", ".rar", ".exe"}
SECRET_EXT = {".jks", ".keystore", ".pem", ".p12", ".pfx", ".key"}
KEY = re.compile(r"AKID[a-zA-Z0-9]{20,}|(?:ghp_|github_pat_)[a-zA-Z0-9_]{20,}|AKIA[A-Z0-9]{16}")
LITERAL = re.compile(r'''(?i)["']?\b[\w]{0,40}(?:secret|password|passwd|private_key|access_key|api_key|token|usersig)[\w]{0,40}["']?\s*(?:=>|=|:)\s*(["'])([^\r\n"']{6,120})\1''')


def extended(path):
    return Path("\\\\?\\" + str(path.resolve())) if os.name == "nt" else path.resolve()


def category(rel):
    p = Path(rel)
    if any(part in SKIP_DIRS for part in p.parts):
        return "exclude-generated"
    if "extend/phpqrcode/cache/" in rel:
        return "exclude-generated"
    if p.name in {"local.properties", ".DS_Store"} or p.name == ".env" or (p.name.startswith(".env.") and p.name not in {".env.example", ".env.template"}) or p.suffix.lower() in SECRET_EXT:
        return "exclude-local-or-credential"
    if rel.startswith("04-tangseng-reference/"):
        return "reference-archive-review"
    if p.suffix.lower() in ARCHIVE_EXT or p.name.startswith("mapping"):
        return "release-asset-review"
    if rel == "FILE_LIST.tsv":
        return "historical-manifest"
    return "source-candidate"


def digest(path):
    h = hashlib.sha256()
    with path.open("rb") as f:
        for block in iter(lambda: f.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()


def scan(path, rel, size):
    if size > 4 * 1024 * 1024 or (path.suffix.lower() not in TEXT_EXT and not path.name.startswith(".env")):
        return [], "not-text-scanned"
    try:
        content = path.read_text(encoding="utf-8-sig")
    except UnicodeError:
        return [], "encoding-review"
    findings = []
    for number, line in enumerate(content.splitlines(), 1):
        rules = set()
        if re.search(r"-----BEGIN (?:RSA |EC |OPENSSH |ENCRYPTED )?PRIVATE KEY-----", line):
            rules.add("private-key-marker")
        if KEY.search(line):
            rules.add("provider-key-pattern")
        # Bound expensive heuristic checks for minified/vendor assets.
        if len(line) <= 4096:
            for match in LITERAL.finditer(line):
                value = match.group(2).lower()
                if any(x in value for x in ("__", "example", "test", "placeholder", "getenv", "env(", "your", "xxxx", "string", "config", "secret", "password", "token", "%", "$", "{", "<", ">", "function", "return")) or " " in value:
                    continue
                rules.add("credential-literal-review")
        if rules:
            findings.append({"path": rel, "line": number, "rules": sorted(rules)})
    return findings, "heuristic-scanned"


def main():
    root = extended(SOURCE)
    rows, findings = [], []
    buckets = defaultdict(lambda: {"files": 0, "bytes": 0})
    print("Inventory started; no credential values will be printed.", flush=True)
    for directory, dirs, files in os.walk(root):
        # Record the exclusion rule rather than enumerating build caches.
        dirs[:] = sorted(d for d in dirs if d not in SKIP_DIRS)
        for name in sorted(files):
            path = Path(directory) / name
            if path.is_symlink():
                raise RuntimeError("Symlink needs explicit review: " + str(path.relative_to(root)))
            rel = path.relative_to(root).as_posix()
            kind = category(rel)
            size = path.stat().st_size
            bucket = buckets[kind]
            bucket["files"] += 1
            bucket["bytes"] += size
            sha = digest(path) if not kind.startswith("exclude-") else ""
            matches, scan_status = scan(path, rel, size) if kind in {"source-candidate", "historical-manifest"} else ([], "excluded-or-archive-not-scanned")
            findings.extend(matches)
            rows.append({"path": rel, "category": kind, "bytes": size, "sha256": sha, "scan": scan_status})
            if len(rows) % 1000 == 0:
                print("Inventoried files:", len(rows), flush=True)
    with (OUT / "FILES.tsv").open("w", encoding="utf-8", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=["path", "category", "bytes", "sha256", "scan"], delimiter="\t")
        writer.writeheader()
        writer.writerows(rows)
    by_path = {row["path"]: row for row in rows}
    old = "02-authoritative-qch-source/企才汇源码交付-20260810/PHP-Web/qicaihui.com/"
    new = "03-php-deployment/backend/"
    a = {k[len(old):]: v for k, v in by_path.items() if k.startswith(old) and v["category"] == "source-candidate"}
    b = {k[len(new):]: v for k, v in by_path.items() if k.startswith(new) and v["category"] == "source-candidate"}
    common = a.keys() & b.keys()
    changed = sorted(k for k in common if a[k]["sha256"] != b[k]["sha256"])
    php = {"identical_count": len(common) - len(changed), "changed": changed, "only_original": sorted(a.keys() - b.keys()), "only_deployment": sorted(b.keys() - a.keys())}
    assets = [row for row in rows if row["category"] in {"release-asset-review", "reference-archive-review"}]
    hashes = defaultdict(list)
    for row in assets:
        hashes[row["sha256"]].append(row["path"])
    report = {"source": SOURCE.name, "counts": dict(buckets), "pruned_directories": sorted(SKIP_DIRS), "php_comparison": php, "identical_asset_groups": [v for v in hashes.values() if len(v) > 1], "credential_review_count": len(findings), "scan_limitations": ["Heuristics are not proof that credentials are absent.", "Archives, binaries and text over 4 MiB were not content-scanned.", "Long minified lines receive provider-key and private-key checks only.", "Build/cache directories are excluded; SSH keys outside the baseline are not read."]}
    for name, data in [("INVENTORY.json", report), ("CREDENTIAL_REVIEW.json", findings), ("RELEASE_ASSETS.json", assets)]:
        (OUT / name).write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"counts": dict(buckets), "credential_review_count": len(findings), "php_changed": len(changed), "php_only_deployment": len(php["only_deployment"])}, ensure_ascii=False), flush=True)


if __name__ == "__main__":
    main()
