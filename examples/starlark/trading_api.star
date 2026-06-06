# A multi-step API check against a token-auth trading-style API:
#   1. Get an auth token.
#   2. List accounts.
#   3. Fetch portfolio for the first account.
#   4. Fetch a quote for XYZ.
# Each step asserts a 200; the chain breaks (probe fails) on the first error.

def probe(target):
    base = "http://%s:%d" % (target.name, target.port)

    # 1. Auth.
    r = http.post(
        url = base + "/api-token-auth/",
        json = {"username": "demo", "password": "demo"},
    )
    assert.http_status(r, 200)
    token = r.json()["token"]
    auth = {"Authorization": "Token " + token}

    # 2. Accounts.
    r = http.get(url = base + "/accounts/", headers = auth)
    assert.http_status(r, 200)
    accounts = r.json()["results"]
    if len(accounts) == 0:
        fail("no accounts returned")
    account_number = accounts[0]["account_number"]

    # 3. Portfolio for first account.
    r = http.get(
        url = base + "/portfolios/%s/" % account_number,
        headers = auth,
    )
    assert.http_status(r, 200)
    equity = r.json()["equity"]
    print("equity for %s: %s" % (account_number, equity))

    # 4. Quote for XYZ.
    r = http.get(url = base + "/quotes/?symbols=XYZ", headers = auth)
    assert.http_status(r, 200)
    quote = r.json()["results"][0]
    if quote["symbol"] != "XYZ":
        fail("expected XYZ, got %s" % quote["symbol"])
