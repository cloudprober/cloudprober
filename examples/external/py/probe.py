#!/usr/bin/env python

import logging

from cloudprober.external import serverutils

def probe(unused_request, reply):
    print("probe() called")
    payload = "success_x 1"
    reply.payload = payload.encode("utf-8")

def main():
    logging.basicConfig()
    serverutils.serve(probe)

if __name__ == "__main__":
    main()