"""Create a reviewable source copy; never changes the live source or pushes Git."""
import csv
import json
import re
import shutil
from pathlib import Path

import inventory

ROOT = inventory.WORKSPACE
DEST = ROOT / "github-ready"
SOURCE_NAME = inventory.SOURCE.name


def main():
    source = inventory.extended(inventory.SOURCE)
    dest = inventory.extended(DEST)
    if not (dest / ".git").is_dir():
        raise RuntimeError("Clone the intended repository before preparing it")
    if (dest / SOURCE_NAME).exists():
        raise RuntimeError("Refusing to overwrite an existing prepared source tree")
    rows = list(csv.DictReader((ROOT / "github-retention/FILES.tsv").open(encoding="utf-8"), delimiter="\t"))
    selected = [r for r in rows if r["category"] == "source-candidate" or r["path"].endswith("/thinkphp/library/think/console/bin/hiddeninput.exe")]
    for row in selected:
        src = source / row["path"]
        if inventory.digest(src) != row["sha256"]:
            raise RuntimeError("Source changed since inventory: " + row["path"])
        dst = dest / SOURCE_NAME / row["path"]
        dst.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(src, dst)
    changes = []

    def edit(rel, fn):
        path = dest / SOURCE_NAME / rel
        original = path.read_text(encoding="utf-8-sig")
        updated = fn(original)
        if updated == original:
            raise RuntimeError("Expected edit did not match: " + rel)
        path.write_text(updated, encoding="utf-8", newline="\n")
        changes.append(rel)

    roots = ["02-authoritative-qch-source/企才汇源码交付-20260810/PHP-Web/qicaihui.com", "03-php-deployment/backend"]
    for root in roots:
        def secondary(text):
            text, count = re.subn(r"(function secondary_password\(\)\s*\{\s*)\$password\s*=\s*'[^']*';[^\n]*\n(?:\s*//[^\n]*\n)?", r"\1$password = getenv('QCH_SECONDARY_PASSWORD_MD5');\n    if (!is_string($password) || !preg_match('/^[a-f0-9]{32}$/i', $password)) {\n        throw new \\RuntimeException('QCH_SECONDARY_PASSWORD_MD5 is not configured');\n    }\n", text, count=1)
            if count != 1:
                raise RuntimeError("Secondary password function missing")
            return text
        edit(root + "/application/common.php", secondary)
        edit(root + "/application/api/model/UserTencent.php", lambda t: re.sub(r"(//\s*\$token\s*=\s*)'[^']+'", r"\1'__SET_LEGACY_CALLBACK_TOKEN__'", t, count=1))
        edit(root + "/.gitignore", lambda t: "\n".join(line for line in t.splitlines() if line.strip() != "thinkphp") + "\n")
    edit(roots[0] + "/public/static/.gitignore", lambda t: "# Preserve product static assets in the source archive.\n.DS_Store\nThumbs.db\n")

    def upload(text):
        text, n1 = re.subn(r"\$secretId\s*=\s*'[^']+';", "$secretId = getenv('TENCENT_SECRET_ID');", text, count=1)
        text, n2 = re.subn(r"\$secretKey\s*=\s*'[^']+';", lambda m: "$secretKey = getenv('TENCENT_SECRET_KEY');\n        if (!$secretId || !$secretKey) {\n            throw new \\RuntimeException('Tencent cloud credentials are not configured');\n        }", text, count=1)
        if n1 != 1 or n2 != 1:
            raise RuntimeError("Expected cloud credential assignments missing")
        return text
    edit(roots[1] + "/application/admin/model/UploadSql.php", upload)

    for name in ["FILES.tsv", "INVENTORY.json", "RELEASE_ASSETS.json", "BASELINE_VERIFICATION.json", "CREDENTIAL_REVIEW.json", "README.md", "inventory.py", "prepare_repository.py"]:
        target = dest / "github-retention" / name
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(ROOT / "github-retention" / name, target)
    shutil.copyfile(ROOT / "github-retention/gitignore.template", dest / ".gitignore")
    shutil.copyfile(ROOT / "github-retention/gitattributes.template", dest / ".gitattributes")
    (dest / "github-retention/ARCHIVE_EDITS.json").write_text(json.dumps({"changed_paths": changes, "reason": "Remove embedded credentials and preserve framework/static resources in Git; original workspace and production are unchanged."}, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"copied_files": len(selected), "edited_files": len(changes)}, ensure_ascii=False))


if __name__ == "__main__":
    main()
