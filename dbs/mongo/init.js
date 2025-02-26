db = db.getSiblingDB('admin');

if(db.auth("root", "root")){
    print("root pass")
}else{
    print("root verification failed")
}

ndb = db.getSiblingDB('lollipop_logs');

ndb.createUser({
    user: "lollipop_logs",
    pwd: "secret",
    roles: [
        { role: "readWrite", db: "lollipop_logs" }
    ]
});
