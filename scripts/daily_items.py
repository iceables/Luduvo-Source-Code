#!/usr/bin/env python3
"""Release 10 catalog items per UTC day: hats, faces, bundles. Never shirts or pants."""
import json
import os
import subprocess
import time
from datetime import datetime, timezone

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DATA = os.path.join(ROOT, "data", "catalog.json")
THUMBS = os.path.join(ROOT, "static", "catalog")
API = "https://api.luduvo.com"
DAILY = 10
ALLOWED = {"hat", "face", "avatar_bundle"}

def curl(url):
    return subprocess.check_output(
        ["curl", "-fsSL", "--max-time", "40", "--socks5-hostname", "127.0.0.1:9050", url],
        stderr=subprocess.DEVNULL,
    )

def load():
    if not os.path.exists(DATA):
        return {"items": [], "by_day": {}}
    with open(DATA, encoding="utf-8") as f:
        return json.load(f)

def save(doc):
    os.makedirs(os.path.dirname(DATA), exist_ok=True)
    tmp = DATA + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(doc, f)
    os.replace(tmp, DATA)

def candidates():
    found = []
    for asset, slot in (("face", None), ("avatar_bundle", None), ("accessory", "hat")):
        offset = 0
        while True:
            url = f"{API}/store?asset_type={asset}&limit=50&offset={offset}"
            page = json.loads(curl(url))
            items = page.get("items") or []
            for item in items:
                kind = item.get("asset_type")
                item_slot = item.get("slot")
                if kind in ("flat_shirt", "flat_pant"):
                    continue
                if kind == "accessory" and item_slot != "hat":
                    continue
                if kind == "face":
                    kind_key = "face"
                elif kind == "avatar_bundle":
                    kind_key = "avatar_bundle"
                elif item_slot == "hat":
                    kind_key = "hat"
                else:
                    continue
                if kind_key not in ALLOWED:
                    continue
                found.append(item["id"])
            offset += 50
            if offset >= int(page.get("total") or 0) or not items:
                break
    return sorted(set(found))

def download_thumb(item_id, url):
    os.makedirs(THUMBS, exist_ok=True)
    dest = os.path.join(THUMBS, f"{item_id}.img")
    if url:
        try:
            data = curl(url)
            if data[:1] not in (b"<", b"{"):
                open(dest, "wb").write(data)
                return f"/static/catalog/{item_id}.img"
        except subprocess.CalledProcessError:
            pass
    return ""

def main():
    doc = load()
    today = datetime.now(timezone.utc).date().isoformat()
    released = doc.setdefault("by_day", {}).get(today, [])
    if len(released) >= DAILY:
        print(f"already released {len(released)} on {today}")
        return
    have = {item["id"] for item in doc.get("items", [])}
    need = DAILY - len(released)
    picked = []
    for item_id in candidates():
        if item_id in have or item_id in released:
            continue
        picked.append(item_id)
        if len(picked) >= need:
            break
    if not picked:
        print("no new hats, faces, or bundles left")
        return
    for item_id in picked:
        detail = json.loads(curl(f"{API}/store/{item_id}"))
        if detail.get("asset_type") in ("flat_shirt", "flat_pant"):
            continue
        remote_thumb = detail.get("thumbnail_url") or ""
        thumb = download_thumb(item_id, remote_thumb)
        if thumb:
            detail["source_thumbnail"] = remote_thumb
            detail["thumbnail_url"] = "https://api.buildoverse.com" + thumb
        detail["released_on"] = today
        variants = ((detail.get("accessory") or {}).get("variants")) or []
        detail["color_schemes"] = [
            {"name": v.get("name"), "hex": v.get("hex"), "index": v.get("index")}
            for v in variants
        ]
        doc.setdefault("items", []).append(detail)
        released.append(item_id)
        time.sleep(0.2)
    doc["by_day"][today] = released
    save(doc)
    print(f"released {len(picked)} on {today}: {picked}")

if __name__ == "__main__":
    main()
