# Probe a Google API using a service-account *self-signed JWT*.
#
# For Google APIs you can skip the OAuth token exchange entirely: mint a JWT
# signed with the service account's private key and send it directly as the
# Bearer credential. Google validates it against the account's public key, so
# there is no round-trip to a token endpoint.
#
# The service-account key material is passed in via `vars` (from the JSON key
# file's `client_email`, `private_key`, and `private_key_id` fields -- see
# gcp_jwt.cfg). This check lists the Cloud Storage buckets for a project, which
# succeeds only if the token authenticates.
#
# Unlike the trading_api example, this one talks to real Google endpoints and
# needs real credentials -- it is illustrative, not part of the mock demo.

def probe(target):
    sa_email = vars.get("sa_email")

    # aud scopes the token to an API; for a self-signed JWT it is the service's
    # base URL. iat/exp are filled in from lifetime (max 1h for Google).
    claims = {
        "iss": sa_email,
        "sub": sa_email,
        "aud": "https://storage.googleapis.com/",
    }

    # kid (the key's private_key_id) lets Google pick the right public key.
    token = jwt.encode(
        claims,
        vars.get("sa_private_key"),
        headers = {"kid": vars.get("sa_private_key_id")},
        lifetime = 3600,
    )

    r = http.get(
        url = "https://storage.googleapis.com/storage/v1/b?project=%s" % vars.get("project"),
        headers = {"Authorization": "Bearer " + token},
    )
    assert.http_status(r, 200)
    print("buckets: %d" % len(r.json().get("items", [])))
