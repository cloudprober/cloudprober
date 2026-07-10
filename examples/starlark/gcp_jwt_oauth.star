# Probe a Google API using a service-account *self-signed JWT* -- the same
# check as gcp_jwt.star, but the JWT is minted and refreshed by cloudprober's
# oauth machinery (oauth_configs with a `jwt` source) instead of the jwt.encode
# builtin. The script just asks for the formatted Authorization header via
# oauth.header("gcp"); minting, caching, and re-minting before expiry all
# happen Go-side.
#
# The service-account key material and JWT claims live in the config
# (gcp_jwt_oauth.cfg), not the script. Like gcp_jwt.star this talks to real
# Google endpoints and needs real credentials -- it is illustrative, not part
# of the mock demo.

def probe(target):
    r = http.get(
        url = "https://storage.googleapis.com/storage/v1/b?project=%s" % vars.get("project"),
        headers = {"Authorization": oauth.header("gcp")},
    )
    assert.http_status(r, 200)
    print("buckets: %d" % len(r.json().get("items", [])))
