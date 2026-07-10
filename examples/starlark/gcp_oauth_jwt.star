# Probe a Google API using a service-account *self-signed JWT*.
#
# For Google APIs you can skip the OAuth token exchange entirely: mint a JWT
# signed with the service account's private key and send it directly as the
# Bearer credential. Here the JWT is minted and refreshed by cloudprober's
# oauth machinery (oauth_configs with a `jwt` source), so the script just asks
# for the formatted Authorization header via oauth.header("gcp"); minting,
# caching, and re-minting before expiry all happen Go-side.
#
# The service-account key material and JWT claims live in the config
# (gcp_oauth_jwt.cfg), not the script. This talks to real Google endpoints and
# needs real credentials -- it is illustrative, not part of the mock demo.

def probe(target):
    r = http.get(
        url = "https://storage.googleapis.com/storage/v1/b?project=%s" % vars.get("project"),
        headers = {"Authorization": oauth.header("gcp")},
    )
    assert.http_status(r, 200)
    print("buckets: %d" % len(r.json().get("items", [])))
