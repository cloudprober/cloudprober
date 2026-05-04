# Example: log in, then fetch a protected endpoint with the returned token.
# Run via:
#   probe {
#     name: "checkout_flow"
#     type: SCRIPT
#     targets { host_names: "api.example.com" }
#     script_probe {
#       source_file: "probes/script/examples/checkout.star"
#     }
#   }

def probe(target):
    r = http.post(
        url = "https://%s/login" % target.name,
        json = {"user": "u", "pass": "p"},
    )
    assert.status(r, 200)
    token = r.json()["token"]

    r = http.get(
        url = "https://%s/cart" % target.name,
        headers = {"Authorization": "Bearer " + token},
    )
    assert.status(r, 200)
