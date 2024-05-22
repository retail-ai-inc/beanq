# beanq

We are heavily working on this project, please stay tuned!


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

