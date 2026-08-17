package beanq

import "testing"

func TestTenantConfigOverridesConnectionsAndKeepsRuntimeSettings(t *testing.T) {
	base := &BeanqConfig{
		Broker: "redis", Redis: Redis{Host: "default-redis", Port: "6379", Password: "old", Prefix: "beanq", PoolSize: 12},
		Mongo:   &Mongo{Host: "default-mongo", Port: "27017", Database: "control", UserName: "old-user", Password: "old-pass", Collections: defaultMongoCollections()},
		History: History{On: true, Storage: "mongo"},
	}
	tenant := tenantConnections{Name: "shop-a", Redis: tenantRedis{Host: "tenant-redis", Port: "6380", Password: "new"}, Mongo: tenantMongo{GCPHost: "tenant-mongo", Port: "27018", DBName: "shop_a", DBUsername: "shop", DBPassword: "secret"}}

	resolved, err := tenantConfig(base, tenant)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Redis.Host != "tenant-redis" || resolved.Redis.Port != "6380" || resolved.Redis.Password != "new" {
		t.Fatalf("unexpected tenant Redis config: %+v", resolved.Redis)
	}
	if resolved.Redis.Prefix != "beanq" || resolved.Redis.PoolSize != 12 {
		t.Fatalf("runtime Redis settings were not inherited: %+v", resolved.Redis)
	}
	if resolved.Mongo.Host != "tenant-mongo" || resolved.Mongo.Database != "shop_a" || resolved.Mongo.UserName != "shop" || resolved.Mongo.Password != "secret" {
		t.Fatalf("unexpected tenant Mongo config: %+v", resolved.Mongo)
	}
	if base.Redis.Host != "default-redis" || base.Mongo.Host != "default-mongo" {
		t.Fatal("tenant resolution mutated the base config")
	}
}

func TestTenantConfigRejectsMissingRequiredTenantConnections(t *testing.T) {
	base := &BeanqConfig{Broker: "redis", Redis: Redis{}, Mongo: &Mongo{}, History: History{On: true, Storage: "mongo"}}
	if _, err := tenantConfig(base, tenantConnections{}); err == nil {
		t.Fatal("expected invalid merged tenant config")
	}
}
