from typing import Collection
import pymongo

client = pymongo.MongoClient("mongodb://root:tvrtestflask@localhost:27017/")

DATABASE_NAME = "ooni"
COLLECTION_NAME = "countries"

def update_db(results):
    db = client[DATABASE_NAME]
    countries = db[COLLECTION_NAME]

    for country in results:
        insert = countries.replace_one({ "_id": country["_id"] }, country, upsert=True)
        print(insert.upserted_id)