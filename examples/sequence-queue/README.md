# Sequence Queue Example

This example verifies the new sequence queue API:

- `PublishNewSequence(channel, topic, orderKey, payload)` publishes interleaved messages for multiple `orderKey` values.
- `ConsumerSequence(channel, topic, handle)` keeps FIFO order inside each `orderKey` list while allowing different `orderKey` values to run in parallel.
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

The publisher does not pass a `seq` field to the queue. Each body contains a visible marker such as `order01-message-01`. The consumer validates the visible marker for each `orderKey` and prints the received message. FIFO execution is provided by the sequence queue itself.

This example publishes 10 `orderKey` values. `order01` has 5 messages, `order03` has 3 messages, and the other order keys have 1 message each. Expected normal output contains increasing body markers per `orderKey`, for example `order01-message-01`, `order01-message-02`, and so on. Messages from different `orderKey` values may appear interleaved.
