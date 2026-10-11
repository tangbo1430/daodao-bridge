# A+ protocol contracts

These fixtures are deliberately small and contain only fields used by the
bridge parser. Replace or extend them with captured, redacted production
frames during the real protocol confirmation stage. Do not commit tokens,
passwords, usersigs or phone numbers.

- `tencent-c2c-after-send.json`: raw Tencent callback accepted by `/cb/tim`.
- `wukong-connect.json`: PC(2) JSON-RPC connect frame.
- `wukong-recv.json`: recv frame that is acknowledged after inbox acceptance.
- `tangseng-sync-request.json`: F incremental sync request shape.
- `PHASE0_CHECKLIST.md`: V1–V12 gate. Unknown fields must be confirmed from
  the real test environment before production behavior is changed.
- `CONTRACT_SAMPLE_TEMPLATE.json`: redacted request/response capture template.
- `A_PLUS_BASELINE_AUDIT.md`: source and deviation audit for the correct archive.

The checked-in JSON files are shape fixtures only. They are not evidence that
the public endpoint or the WebSocket route is correct. Replace or extend them
with redacted captures and update the checklist evidence column.
