import json
import logging
import requests
import datetime
from requests.models import PreparedRequest
from country.country import get_countries
from db.db import update_db

logging.basicConfig(filename='../logs/ooni.log',
                    filemode='a',
                    format='%(asctime)s,%(msecs)d %(name)s %(levelname)s %(message)s',
                    datefmt='%Y-%m-%d %H:%M:%S',
                    level=logging.DEBUG)
logging.getLogger().addHandler(logging.StreamHandler())

OONI_API_BASE = "https://api.ooni.io/api/_"

OONI_RESULTS_PER_PAGE = 1000
OONI_PROBE_THRESHOLD = 50

# Builds a request to OONI using dictionary of query parameters
def get_ooni_json(path: str, query: dict = {}):
    url = f"{OONI_API_BASE}{path}"

    req = PreparedRequest()
    req.prepare_url(url, query)

    logging.debug(f"GET request to OONI with URL {req.url}")

    resp = requests.get(req.url)
    return resp.json()

# Get the ASNs with the most probes (e.g. Telstra in Australia)
def get_asn(country: str):
    asn = get_ooni_json("/website_networks", {
        "probe_cc": country
    })

    if not asn["results"]: # No ASN
        print(f"Country {country} has no ASN")
        return None

    # OONI will likely give the max count ASN, but just to be sure
    asn_max = max(asn["results"], key=lambda k:k["count"])
    return asn_max["probe_asn"]

def get_country(country: dict, asn: str):
    results = True
    offset = 0

    sites = []

    while(results):
        res = get_ooni_json("/website_urls", {
            "limit": OONI_RESULTS_PER_PAGE, # Returned results per page
            "offset": offset,
            "probe_asn": asn,
            "probe_cc": country["alpha2Code"]
        })

        if not res["results"]: # No more results
            results = False

        for site in res["results"]:
            if(site["total_count"] >= OONI_PROBE_THRESHOLD):
                sites.append(build_site_model(site))

        offset = offset + OONI_RESULTS_PER_PAGE

        country["asnProbed"] = asn

    return {
        "_id": country["alpha2Code"], # Used by MongoDb for the ID
        "country": country,
        "sites": sites
    }

def build_site_model(site: dict):
    url = site["input"]

    blocked = (((site["anomaly_count"] + site["failure_count"]) >= (site["total_count"] * 0.7)) or
                (site["confirmed_count"] >= (site["total_count"] * 0.5)))

    del site["input"]

    return {
        "site": url,
        "lastChecked": datetime.datetime.utcnow().replace(tzinfo=datetime.timezone.utc).isoformat(),
        "blocked": blocked,
        "confidence": site["confirmed_count"]/site["total_count"],
        "result": site
    }

# Triggered every day?
if __name__ == "__main__":
    countries = get_countries()
    results = []

    for country in countries:
        country_code = country["alpha2Code"]

        print(country_code)

        asn = get_asn(country_code)
        if asn:
            c = get_country(country, asn)

            with open(f"results/{country_code}.json", "w") as o:
                json.dump(c, o, indent=4)

            results.append(c)
    update_db(results)