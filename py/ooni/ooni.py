import logging
import requests
from requests.models import PreparedRequest

OONI_API_BASE = "https://api.ooni.io/api/_"

# Builds a request using dictionary of query parameters
def get_ooni_json(path: str, query: dict = {}):
    url = f"{OONI_API_BASE}{path}"

    req = PreparedRequest()
    req.prepare_url(url, query)

    logging.debug(f"GET request to OONI with URL {req.url}")

    print(req.url)

    resp = requests.get(req.url)
    return resp.json()

# Get the ASNs with the most probes (e.g. Telstra in Australia)
def get_asn(country: str):
    asn = get_ooni_json("/website_networks", {
        "probe_cc": country
    })

    # OONI will likely give the max count ASN, but just to be sure
    asn_max = max(asn["results"], key=lambda k:k["count"])
    return asn_max["probe_asn"]



if __name__ == "__main__":
    print(get_asn("AU"))