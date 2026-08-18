package beanq

import (
	"context"
	"fmt"
	"strings"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type tenantConnections struct {
	Name  string      `bson:"name"`
	Redis tenantRedis `bson:"redis"`
	Mongo tenantMongo `bson:"mongo"`
}

type tenantRedis struct {
	Host     string    `bson:"host"`
	GCPHost  string    `bson:"gcpHost"`
	Port     string    `bson:"port"`
	Password string    `bson:"password"`
	SSL      tenantSSL `bson:"ssl"`
}

type tenantMongo struct {
	Host       string    `bson:"host"`
	GCPHost    string    `bson:"gcpHost"`
	Port       string    `bson:"port"`
	DBName     string    `bson:"dbName"`
	DBUsername string    `bson:"dbUsername"`
	DBPassword string    `bson:"dbPassword"`
	SSL        tenantSSL `bson:"ssl"`
}

type tenantSSL struct {
	On                bool   `bson:"on"`
	VerifyCertificate bool   `bson:"verifyCertificate"`
	HotReload         bool   `bson:"hotReload"`
	CertFile          string `bson:"certFile"`
}

func resolveTenantConfig(ctx context.Context, base *BeanqConfig, tenantCode string) (ResolvedConfig, error) {
	code := strings.TrimSpace(tenantCode)
	if code == "" {
		return ResolvedConfig{}, berror.ErrInvalidConfig.WithMessage("tenant code is required")
	}
	if base == nil || base.Mongo == nil {
		return ResolvedConfig{}, berror.ErrInvalidConfig.WithMessage("default mongo config is required for tenant lookup")
	}
	client, err := newMongoClient(ctx, base.Mongo)
	if err != nil {
		return ResolvedConfig{}, fmt.Errorf("connect default mongo: %w", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	var tenant tenantConnections
	collection := base.Mongo.collectionName("tenant", "tenants")
	err = client.Database(base.Mongo.Database).Collection(collection).FindOne(ctx, bson.M{"code": code}).Decode(&tenant)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return ResolvedConfig{}, fmt.Errorf("tenant %q not found", code)
		}
		return ResolvedConfig{}, fmt.Errorf("query tenant %q: %w", code, err)
	}
	return tenantConfig(base, tenant)
}

func tenantConfig(base *BeanqConfig, tenant tenantConnections) (ResolvedConfig, error) {
	config := cloneBeanqConfig(*base)
	if host := tenantHost(tenant.Redis.Host, tenant.Redis.GCPHost); host != "" {
		config.Redis.Host = host
	}
	if tenant.Redis.Port != "" {
		config.Redis.Port = tenant.Redis.Port
	}
	config.Redis.Password = tenant.Redis.Password
	config.Redis.SSL.On = tenant.Redis.SSL.On
	config.Redis.SSL.Verify = tenant.Redis.SSL.VerifyCertificate
	config.Redis.SSL.HotReload = tenant.Redis.SSL.HotReload
	config.Redis.SSL.CAFile = tenant.Redis.SSL.CertFile

	if host := tenantHost(tenant.Mongo.Host, tenant.Mongo.GCPHost); host != "" {
		config.Mongo.Host = host
	}
	if tenant.Mongo.Port != "" {
		config.Mongo.Port = tenant.Mongo.Port
	}
	if tenant.Mongo.DBName != "" {
		config.Mongo.Database = tenant.Mongo.DBName
	}
	config.Mongo.UserName = tenant.Mongo.DBUsername
	config.Mongo.Password = tenant.Mongo.DBPassword
	config.Mongo.SSL.On = tenant.Mongo.SSL.On
	config.Mongo.SSL.Verify = tenant.Mongo.SSL.VerifyCertificate
	config.Mongo.SSL.HotReload = tenant.Mongo.SSL.HotReload
	config.Mongo.SSL.CAFile = tenant.Mongo.SSL.CertFile
	return config.Resolve()
}

func tenantHost(host, fallback string) string {
	if strings.TrimSpace(host) != "" {
		return host
	}
	return fallback
}
