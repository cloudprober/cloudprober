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

"""
This module provides utilities to work with the cloudprober's external probe.
"""

import io
import logging
import sys
import threading
import queue
import time
from typing import Callable

from google.protobuf.message import Message, DecodeError

import cloudprober.external.server_pb2 as serverpb

class StopThread(Exception):
    pass

class Context:
    """
    Context class to handle stopping threads based on a given attribute.
    """

    def __init__(self, stop_attr=None):
        self.stop_attr = stop_attr

    def __enter__(self):
        if not self.stop_attr:
            return
        if getattr(threading.current_thread(), self.stop_attr, False):
            raise StopThread("Stopping as per the thread attribute")

    def __exit__(self, exc_type, exc_value, traceback):
        pass

def _read_payload(r: io.BufferedReader, ctx: Context = Context()) -> bytes:
    """
    Read the payload from the given BufferedReader.

    Args:
        r (io.BufferedReader): The BufferedReader to read from.
        ctx (Context, optional): The context to handle stopping the thread. Defaults to Context().

    Returns:
        bytes: The payload read from the BufferedReader.
    """
    # header format is: "\nContent-Length: %d\n\n"
    prefix = b"Content-Length: "
    line = ""
    length = 0
    err = None

    # Read lines until header line is found
    while True:
        with ctx:
            try:
                line = r.readline()
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

    r.read(1)  # Read the newline after the header.
    return r.read(length)
    
def _read_probe_request(r: io.BufferedReader, ctx: Context = Context()) -> serverpb.ProbeRequest:
    """
    Read the probe request from the given BufferedReader.

    Args:
        r (io.BufferedReader): The BufferedReader to read from.
        ctx (Context, optional): The context to handle stopping the thread. Defaults to Context().

    Returns:
        serverpb.ProbeRequest: The parsed ProbeRequest object.
    """
    payload = _read_payload(r, ctx)
    if not payload:
        return None
    req = serverpb.ProbeRequest()
    try:
        req.ParseFromString(payload)
    except DecodeError as e:
        logging.error("Error decoding probe request: %s", e)
        return None
    return req

def _write_message(pb: Message, w: io.BufferedWriter) -> None:
    """
    Write the given protobuf message to the given writer.

    Args:
        pb (Message): The protobuf message to write.
        w: The writer to write to.
    """
    buf = pb.SerializeToString()
    try:
        w.write(b"Content-Length: %d\n\n" % len(buf))
        w.write(buf)
        w.flush()
    except Exception as e:
        logging.error(f"Error writing message: {str(e)}")

def serve(probe_func: Callable, stdin=sys.stdin.buffer, stdout=sys.stdout.buffer, stderr=sys.stderr, ctx: Context = Context()):
    """
    Serve the probe requests by reading from stdin and writing to stdout.

    Args:
        probe_func (Callable): The function to handle the probe requests.
        stdin (io.BufferedIOBase, optional): The input stream to read the probe requests from. Defaults to sys.stdin.buffer.
        stdout (io.BufferedIOBase, optional): The output stream to write the probe replies to. Defaults to sys.stdout.buffer.
        stderr (io.BufferedIOBase, optional): The error stream to write the error logs to. Defaults to sys.stderr.
        ctx (Context, optional): The context to handle stopping the thread. Defaults to Context().
    """
    replies_queue = queue.Queue()

    # Write replies to stdout. These are not required to be in-order.
    def write_replies():
        while True:
            with ctx:
                reply = replies_queue.get(block=True)
                if reply is None:
                    continue
                _write_message(reply, stdout)
                    

    threading.Thread(target=write_replies, daemon=True).start()

    # Read requests from stdin, and dispatch probes to service them.
    while True:
        with ctx:
            request = _read_probe_request(stdin, ctx)
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