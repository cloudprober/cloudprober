import io
import logging
import os
import threading
import time
import unittest
from unittest.mock import patch

import cloudprober.external.server_pb2 as serverpb
from cloudprober.external.serverutils import (
    _read_payload,
    _read_probe_request,
    _write_message,
    serve,
    Context,
    StopThread,
)

class ExcThread(threading.Thread):
    def run(self):
        try:
            super().run()
        except StopThread:
            pass

class TestServerUtils(unittest.TestCase):

    test_request = serverpb.ProbeRequest()
    test_request.request_id = 10
    test_request.time_limit = 1000

    test_reply = serverpb.ProbeReply()
    test_reply.request_id = 20
    test_reply.payload = b"Test Payload"

    def test_read_payload(self):
        payload = b"Hello, World!"
        buf = io.BufferedReader(io.BytesIO(b"\nContent-Length: 13\n\nHello, World!"))
        result = _read_payload(buf)
        self.assertEqual(result, payload)

    def test_read_probe_request(self):
        # Test case for read_probe_request function
        buf = io.BytesIO()
        _write_message(self.test_request, buf)
        buf.seek(0)
        result = _read_probe_request(buf)
        self.assertEqual(result.request_id, self.test_request.request_id)
        self.assertEqual(result.time_limit, self.test_request.time_limit)

    def test_write_message(self):
        # Test case for write_message function
        buf = io.BytesIO()
        _write_message(self.test_reply, buf)
        buf.seek(0)
        payload = _read_payload(buf)
        reply = serverpb.ProbeReply()
        reply.ParseFromString(payload)
        self.assertEqual(reply.request_id, self.test_reply.request_id)
        self.assertEqual(reply.payload, self.test_reply.payload)

    @unittest.skipIf(os.name != "posix", "Skipping test on non-Unix systems")    
    def test_serve(self):
        def probe_func(request, reply):
            reply.request_id = request.request_id
            reply.payload = "Hello, World!"

        stdin_read, stdin_write = os.pipe()
        stdout_read, stdout_write = os.pipe()

        ctx = Context("stop")
        with open(stdin_read, "rb") as stdin , open(stdout_write, "wb") as stdout:
            t = ExcThread(target=serve, args=(probe_func, stdin, stdout, io.BytesIO(), ctx), daemon=True)
            t.start()
            time.sleep(0.1)  # Wait for the serve function to start processing

            with open(stdin_write, "wb") as w:
                _write_message(self.test_request, w)
            
            with open(stdout_read, "rb") as r:
                payload = _read_payload(r)

            reply = serverpb.ProbeReply()
            reply.ParseFromString(payload)
            self.assertEqual(reply.request_id, 10)
            self.assertEqual(reply.payload, "Hello, World!")
            time.sleep(1)  # Wait for the serve function to exit
            t.stop = True
        
if __name__ == "__main__":
    unittest.main()
