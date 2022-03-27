import requests
from requests.models import PreparedRequest

COUNTRY_API_BASE = "https://restcountries.com/v2"

# Example response:
# [{
#     "name": "Australia",
#     "alpha2Code": "AU",
#     "alpha3Code": "AUS",
#     "population": 25687041,
#     "flags": {
#         "svg": "https://flagcdn.com/au.svg",
#         "png": "https://flagcdn.com/w320/au.png"
#     },
#     "independent": false
# }
# ...
# ]
def get_countries():
    url = f"{COUNTRY_API_BASE}/all"

    req = PreparedRequest()
    req.prepare_url(url, {
        "fields": "name,alpha2Code,alpha3Code,population,flags"
    })

    return requests.get(req.url).json()