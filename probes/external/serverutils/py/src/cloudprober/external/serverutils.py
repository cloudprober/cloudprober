# Copyright 2024 The Cloudprober Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# serverutils provides utilities to work with the cloudprober's external probe.

import io
import logging
import sys
import threading
import queue
import time
from typing import Callable

from google.protobuf.message import Message, DecodeError

import cloudprober.external.server_pb2 as serverpb

def _read_payload(r: io.BufferedReader) -> bytes:
    # header format is: "\nContent-Length: %d\n\n"
    prefix = b"Content-Length: "
    line = ""
    length = 0
    err = None

    # Read lines until header line is found
    while True:
        try:
            line = r.buffer.readline()
        except UnicodeDecodeError as e:
            continue
        if line.startswith(prefix):
            try:
                length = int(line[len(prefix):])
            except ValueError as e:
                err = e
            break
    if err:
        logging.error("Error reading payload length: %s", err)
        return b""
    
    if length <= 0:
        return b""

    r.buffer.read(1)  # Read the newline after the header.
    return r.buffer.read(length)
    
def _read_probe_request(r: io.BufferedReader) -> serverpb.ProbeRequest:
    payload = _read_payload(r)
    if not payload:
        return None
    req = serverpb.ProbeRequest()
    try:
        req.ParseFromString(payload)
    except DecodeError as e:
        logging.error("Error decoding probe request: %s", e)
        return None
    return req

def _write_message(pb: Message, w):
    buf = pb.SerializeToString()
    try:
        w.buffer.write(b"Content-Length: %d\n\n" % len(buf))
        w.buffer.write(buf)
        w.flush()
    except Exception as e:
        raise Exception(f"Failed writing response: {str(e)}")
    
def serve(probe_func: Callable, stdin=sys.stdin, stdout=sys.stdout, stderr=sys.stderr):
    replies_queue = queue.Queue()

    # Write replies to stdout. These are not required to be in-order.
    def write_replies():
        while True:
            reply = replies_queue.get(block=True)
            if reply is None:
                continue
            if _write_message(reply, stdout) is not None:
                sys.exit(1)

    threading.Thread(target=write_replies, daemon=True).start()

    # Read requests from stdin, and dispatch probes to service them.
    while True:
        request = _read_probe_request(stdin)
        if request is None:
            sys.exit(1)

        def handle_request():
            reply = serverpb.ProbeReply()
            reply.request_id = request.request_id
            done = threading.Event()
            timeout = time.time() + request.time_limit / 1000

            def probe():
                probe_func(request, reply)
                done.set()

            threading.Thread(target=probe, daemon=True).start()

            if done.wait(timeout - time.time()):
                replies_queue.put(reply)
            else:
                logging.error(f"Timeout for request {reply.request_id}", file=stderr)

        threading.Thread(target=handle_request, daemon=True).start()