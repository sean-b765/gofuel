"""Fetch NT fuel outlet prices from myfuelnt.nt.gov.au (endpoint #4).

The /Home/Results page embeds the full dataset for every NT outlet in a
hidden <input id="serverJson">. One GET returns all stations with
per-fuel current and scheduled prices.
"""

import html
import json
import re
import sys
import urllib.parse
import urllib.request

BASE = "https://myfuelnt.nt.gov.au"
SERVER_JSON_RE = re.compile(
    r'<input type="hidden" id="serverJson" value="(.*?)"', re.S
)


def fetch_results(suburb="DARWIN CITY (0800)", suburb_id=1, region_id=0,
                  fuel_code="ALLD", brand="All Brands"):
    qs = urllib.parse.urlencode({
        "Suburb": suburb,
        "SuburbId": suburb_id,
        "RegionId": region_id,
        "FuelCode": fuel_code,
        "BrandIdentifier": brand,
        "searchOptions": "suburbPostcode",
    })
    req = urllib.request.Request(
        f"{BASE}/Home/Results?{qs}",
        headers={"User-Agent": "Mozilla/5.0"},
    )
    with urllib.request.urlopen(req) as r:
        body = r.read().decode()
    m = SERVER_JSON_RE.search(body)
    if not m:
        raise RuntimeError("serverJson not found in response")
    return json.loads(html.unescape(m.group(1)))


def flatten_outlets(data):
    rows = []
    for o in data["FuelOutlet"]:
        base = {
            "outlet_id": o["FuelOutletId"],
            "outlet_identifier": o["FuelOutletIdentifier"],
            "name": o["OutletName"],
            "brand": o["OutletBrandIdentifier"],
            "address": o["Address"],
            "suburb": o["Suburb"],
            "postcode": o["Postcode"],
            "state": o["OutletState"],
            "region_id": o["RegionId"],
            "latitude": o["Latitude"],
            "longitude": o["Longitude"],
            "is_active": o["IsActive"],
            "web_address": o.get("WebAddress"),
        }
        if not o["AvailableFuels"]:
            rows.append({**base, "fuel_code": None, "price": None,
                         "price_scheduled": None,
                         "is_available": None,
                         "is_available_scheduled": None})
            continue
        for f in o["AvailableFuels"]:
            rows.append({
                **base,
                "fuel_code": f["FuelCode"],
                "price": f["Price"],
                "price_scheduled": f["PriceScheduled"],
                "is_available": f["isAvailable"],
                "is_available_scheduled": f["isAvailableScheduled"],
            })
    return rows


def main():
    data = fetch_results()
    rows = flatten_outlets(data)
    json.dump(rows, sys.stdout, indent=2)
    print(f"\n\n{len(rows)} rows from {len(data['FuelOutlet'])} outlets",
          file=sys.stderr)


if __name__ == "__main__":
    main()