package beanq

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mongodb"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// pass configuration information through the flags parameter
const cmdConfigKeyName = "config"

var (
	rootCmd = &cobra.Command{
		Use:   "beanq",
		Short: "",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}
	migrationCmd = &cobra.Command{
		Use:   "migration",
		Short: "",
		Run:   migration,
	}
)

func init() {
	runCmd.AddCommand(migrationCmd)
	rootCmd.AddCommand(runCmd)
}

// migration database schema for logs and configs
func migration(cmd *cobra.Command, args []string) {

	if changed := cmd.Flags().Changed(cmdConfigKeyName); changed {
		logger.New().Panic("Prohibit modifying [config] parameter values")
	}

	conf, err := parseConfig(cmd.Flags())
	if err != nil {
		logger.New().Panic(err)
	}
	action, err := cmd.Flags().GetString("action")
	if err != nil {
		logger.New().Panic(err)
	}
	//
	mCtx := NewMigrateContext(NewMongoMigrater(conf, action, migrationsFS))
	mCtx.Execute()

	// if have mysql migrater
	//mCtx.Set(NewMySqlMigrater())
	//mCtx.Execute()

	logger.New().Info("migration successful")
	os.Exit(0)
}

type Migrater interface {
	Collections() map[string]map[string]Collection
	SortedVersions(collections map[string]map[string]Collection) ([]string, error)
	MigrationInstance() (*migrate.Migrate, error)
}

type MongoMigrater struct {
	client *mongo.Client
	Config *BeanqConfig
	action string
	fsys   embed.FS
}

var _ Migrater = (*MongoMigrater)(nil)

func NewMongoMigrater(beanqConfig *BeanqConfig, action string, fsys embed.FS) *MongoMigrater {

	client, err := newMongoClient(context.Background(), beanqConfig.Mongo)
	if err != nil {
		panic(err)
	}
	return &MongoMigrater{
		client: client,
		Config: beanqConfig,
		action: action,
		fsys:   fsys,
	}
}

func (t *MongoMigrater) Collections() map[string]map[string]Collection {

	collections := make(map[string]Collection)
	if t.Config.History.Storage == "mongo" {
		//nolint:staticcheck,qf1008 //enhance readability
		collections = t.Config.Mongo.Collections
	}
	if t.Config.WorkFlow.Storage != "mongo" {
		delete(collections, "workflow")
	}

	data := map[string]map[string]Collection{
		"mongo": collections,
	}
	return data
}

func (t *MongoMigrater) SortedVersions(collections map[string]map[string]Collection) ([]string, error) {

	var files []string

	existInCollections := func(collectionPath string) bool {
		for _, vals := range collections {
			for _, collection := range vals {
				name := strings.Join([]string{"beanq.", collection.Name, ".", t.action}, "")
				if strings.Contains(collectionPath, name) {
					return true
				}
			}
		}
		return false
	}

	err := fs.WalkDir(t.fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !existInCollections(path) {
			return err
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return extractVersion(files[i]) < extractVersion(files[j])
	})
	return files, nil
}

func (t *MongoMigrater) MigrationInstance() (*migrate.Migrate, error) {

	mongoCfg := t.Config.Mongo

	driver, err := mongodb.WithInstance(t.client, &mongodb.Config{DatabaseName: mongoCfg.Database})
	if err != nil {
		return nil, err
	}
	sourceDriver, err := iofs.New(migrationsFS, "migrations/mongo")
	if err != nil {
		return nil, err
	}

	mig, err := migrate.NewWithInstance("iofs", sourceDriver, "mongo", driver)
	if err != nil {
		return nil, err
	}

	return mig, nil
}

type MigrateContext struct {
	migrater Migrater
}

func NewMigrateContext(migrater Migrater) *MigrateContext {
	return &MigrateContext{migrater: migrater}
}

func (t *MigrateContext) Set(migrater Migrater) {
	t.migrater = migrater
}

func (t *MigrateContext) Execute() {

	collections := t.migrater.Collections()

	availableVersions, err := t.migrater.SortedVersions(collections)
	if err != nil {
		log.Fatalf("failed to parse version: %v", err)
	}
	fmt.Println("=== available migration versions ===")
	for _, v := range availableVersions {
		fmt.Println(v)
	}

	m, err := t.migrater.MigrationInstance()
	if err != nil {
		log.Fatalf("failed to create migration instance: %v", err)
	}
	currentVersion, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		log.Fatalf("failed to obtain the current version: %v", err)
	}
	fmt.Println("--------------------------------------------------------------------")
	if dirty {
		log.Fatalf("database version is [dirty], please repair manually")
	}
	if err == migrate.ErrNilVersion {
		currentVersion = 0
	}
	fmt.Printf("current database version: %d\n", currentVersion)

	var pending []uint64
	for _, v := range availableVersions {
		if extractVersion(v) > uint64(currentVersion) {
			pending = append(pending, extractVersion(v))
		}
	}
	if len(pending) == 0 {
		fmt.Println("it is already the latest version, no migration needed")
		os.Exit(0)
	}
	fmt.Println("--------------------------------------------------------------------")
	fmt.Printf("pending application version: %v\n", pending)

	for _, target := range pending {
		start := time.Now()
		if err := m.Migrate(uint(target)); err != nil {
			if err == migrate.ErrNoChange {
				continue
			}
			log.Fatalf("version %d failed: %v", target, err)
		}
		duration := time.Since(start)
		fmt.Printf("=== successfully used version %d，run times: %v ===\n", target, duration)
	}
}

//go:embed migrations
var migrationsFS embed.FS

func parseConfig(flags interface{ GetString(string) (string, error) }) (*BeanqConfig, error) {

	jsonStr, err := flags.GetString(cmdConfigKeyName)
	if err != nil {
		return nil, err
	}
	var cfg BeanqConfig
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func newMongoClient(ctx context.Context, config *Mongo) (*mongo.Client, error) {

	mport := strings.TrimLeft(config.Port, ":")
	mport = fmt.Sprintf(":%s", mport)
	uri := strings.Join([]string{"mongodb://", config.Host, mport}, "")

	opts := options.Client().ApplyURI(uri).
		SetConnectTimeout(config.ConnectTimeOut).
		SetMaxPoolSize(config.MaxConnectionPoolSize).
		SetMaxConnIdleTime(config.MaxConnectionLifeTime)

	if config.UserName != "" && config.Password != "" {
		auth := options.Credential{
			AuthSource: config.Database,
			Username:   config.UserName,
			Password:   config.Password,
		}
		opts.SetAuth(auth)
	}

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return client, nil
}

func extractVersion(filename string) uint64 {

	base := filepath.Base(filename)
	parts := strings.SplitN(base, "_", 2)
	if len(parts) == 0 {
		return 0
	}
	v, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// Execute commands
func Execute() error {

	if err := rootCmd.Execute(); err != nil {
		return err
	}
	return nil
}
