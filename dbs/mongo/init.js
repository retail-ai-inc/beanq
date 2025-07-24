db = db.getSiblingDB('beanq_logs');

db.createUser({
    user: "beanq",
    pwd: "secret",
    roles: [
        { role: "readWrite", db: "beanq_logs" },
        { role: "dbAdmin", db: "beanq_logs"}
    ]
});
