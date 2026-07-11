# Sequence Queue Example

This example verifies the new sequence queue API:

- `PublishNewSequence(channel, topic, customerId, payload)` publishes interleaved messages for multiple `customerId` values.
- `ConsumerSequence(channel, topic, handle)` keeps FIFO order inside each `customerId` list while allowing different `customerId` values to run in parallel.
- `sequenceQueuePartitions` fixes the number of Redis Cluster scheduler streams. It must remain identical across publishers and consumers after the queue is created.
- Delivery is at-least-once: handlers should be idempotent by message ID. A worker crash may repeat the current message, but a stale owner cannot advance the Redis queue state.

Run the consumer first:

```bash
go run ./examples/sequence-queue/consumer
```

Then publish test messages from another terminal:

```bash
go run ./examples/sequence-queue/publisher
```

The publisher does not pass a `seq` field to the queue. The message body only contains a visible publish marker such as `customer-a-message-01` so the example can assert FIFO behavior. The queue ordering itself comes from `RPUSH` into the per-`customerId` list and consuming the list head after the previous message finishes.

If a message for the same `customerId` is consumed out of FIFO order, the handler returns an error like:

```text
out of order: customerId=customer-a expectedBody=customer-a-message-03 gotBody=customer-a-message-04
```

This example publishes 10 `customerId` values. `customer01` has 5 messages, `customer03` has 3 messages, and the other customer IDs have 1 message each. Expected normal output contains increasing body markers per `customerId`, for example `customer01-message-01`, `customer01-message-02`, and so on. Messages from different `customerId` values may appear interleaved.


To verify that a failed head message is logged and later messages for the same `customerId` continue in FIFO order, run the consumer with:

```bash
BEANQ_SEQUENCE_FAIL_BODY=customer01-message-01 go run ./examples/sequence-queue/consumer
```

Then run the publisher normally. The consumer should still handle `customer01-message-02` through `customer01-message-05` after logging the simulated failure for `customer01-message-01`.
