# Beanq

Beanq is a message queue system developed based on the Redis data type ```Stream```, which currently supports ```normal queues```, ```delay queues```, and ```sequence queues```.

## Notice 
To ensure data safety, it is recommended to enable AOF persistence. For the differences between AOF and RDB, please refer to the official documentation [Redis persistence](https://redis.io/docs/latest/operate/oss_and_stack/management/persistence/)

## Process
1. **Normal Queue**\
    ![Alt](/ui/static/images/normal.png#pic_center=80*80)
    >When messages are published to the queue, they are consumed immediately. Concurrently, a child coroutine is initiated to monitor for any dead-letter messages. Should dead-letter messages be detected, they are directly moved to a log queue to facilitate further handling or analysis.\
    In this context:\
        **1.1**  "published to the queue" means that messages are added to the queue for processing.\
        **1.2**  "consumed directly" implies that these messages are processed as soon as they become available.\
        **1.3**  "subprocess coroutine" refers to an auxiliary concurrent process that runs alongside the main process.\
        **1.4**  "dead-letter messages" are messages that could not be delivered or processed successfully and have been moved to a special queue for handling.\
        **1.5**  "log queue" is a designated queue where dead-letter messages are stored for later examination or retry.
2.  **Delay Queue**\
    ![Alt](/ui/static/images/delay.png#pic_center=80*80)
    >The delay queue system supports messages with priority levels. The format for storing these messages is ```1734399237.999```, where:\
    **2.1** The preceding segment (e.g., 1734399237) indicates the Unix timestamp in seconds, defining when the message should be processed.\
    **2.2** The succeeding segment (e.g., .999) represents the priority level of the message.\
    **2.3** The system allows for a maximum priority level of 999. Messages with a higher numerical value in the priority segment will be given precedence and therefore consumed before those with lower values. This mechanism ensures that critical or time-sensitive messages are handled as a priority within the delay queue framework.

3. **Sequence Queue** \
   ![Alt](/ui/static/images/sequence.png#pic_center=80*80)
   >In synchronous messaging, the status of a message is synchronized to a Redis hash at every stage of the message-sending process.\
    This means that as the message progresses through different stages (e.g., creation, sending, delivery), its status is updated in a Redis hash. \
    **3.1** The client then performs synchronous checks against this Redis hash to retrieve the current status of the message. Based on the information retrieved from the Redis hash, the client can then return or provide feedback about the message accordingly.\
    **3.2** This approach ensures that the client always has the most up-to-date information regarding the message's status, allowing for immediate responses based on the latest state of the message within the system.
## Example Explanation

Start and enter the container.
```bash
docker compose up -d --build

docker exec -it beanq-example bash
```

delay example:
```bash
make delay
```

normal example:
```bash
make normal
```

sequential example:
```bash
make sequential
```

When you want to exit the container, please remember to execute the clean command, as env.json needs to be restored.
```bash
make clean
```

