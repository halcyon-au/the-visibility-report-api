import logging
from flask import Flask, jsonify
from flasgger import Swagger
import requests

logging.basicConfig(filename='./logs/api.log',
                    filemode='a',
                    format='%(asctime)s,%(msecs)d %(name)s %(levelname)s %(message)s',
                    datefmt='%Y-%m-%d %H:%M:%S',
                    level=logging.DEBUG)
logging.getLogger().addHandler(logging.StreamHandler())

app = Flask(__name__)
swagger = Swagger(app)

API_VER = 1
API_BASE = f"/api/v{API_VER}"

@app.route('/')
def index():
    return "<h1>The Visibility Report API</h1><br><br\> \
            <a href='#'>Website</a>"

@app.route(f"{API_BASE}/hb", methods=['GET'])
def heartbeat():
    resp = requests.get("https://ooni.org")

    return jsonify({
        "result": "API up 😎",
        "ooniReachable": resp.status_code == 200
    })

