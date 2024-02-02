package schema

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/tern/migrate"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	pg "github.com/remiges-tech/crux/db"
	"github.com/remiges-tech/logharbour/logharbour"
)

var r *gin.Engine
var versionTable string = "schema_version_non_default"

func TestMain(m *testing.M) {

	// Initialize Docker pool to insure it close at the end
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to Docker: %v", err)
	}
	fmt.Println("Initialize Docker pool")

	// Deploy the Postgres container.
	resource, databaseUrl, err := deployPostgres(pool)
	if err != nil {
		log.Fatalf("Could not start resource: %v", err)
	}
	fmt.Println("Deploy the Postgres container")

	ternMigrate(databaseUrl)
	fmt.Println("tern migrate")

	// Register routes.
	r, err = registerRoutes(pool, databaseUrl)
	if err != nil {
		log.Fatalf("Could not start resource: %v", err)
	}
	fmt.Println("Register routes")
	// Run the tests.
	exitCode := m.Run()

	// Exit with the appropriate code.
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}
	fmt.Println("Exit with the appropriate code")

	os.Exit(exitCode)

}

// deployPostgres builds and runs the Postgres container.
func deployPostgres(pool *dockertest.Pool) (*dockertest.Resource, string, error) {
	// pulls an image, creates a container
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "11",
		Env: []string{
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_USER=root",
			"POSTGRES_DB=crux",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseUrl := fmt.Sprintf("postgres://root:postgres@%s/crux?sslmode=disable", hostAndPort)

	log.Println("Connecting to database on url: ", databaseUrl)

	resource.Expire(120) // Tell docker to hard kill the container in 120 seconds
	pool.MaxWait = 120 * time.Second

	// Ensure the Postgres container is ready to accept connections.
	if err = pool.Retry(func() error {
		db, err := sql.Open("postgres", databaseUrl)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	return resource, databaseUrl, nil

}

// registerRoutes register and runs.
func registerRoutes(pool *dockertest.Pool, databaseUrl string) (*gin.Engine, error) {
	// router
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// logger setup
	fallbackWriter := logharbour.NewFallbackWriter(log.Writer(), os.Stdout)
	lctx := logharbour.NewLoggerContext(logharbour.Info)
	l := logharbour.NewLogger(lctx, "crux", fallbackWriter)

	file, err := os.Open("./../../errortypes.yaml")
	if err != nil {
		log.Fatalf("Failed to open error types file: %v", err)
	}
	defer file.Close()
	wscutils.LoadErrorTypes(file)

	// Database connection
	query, err := pg.NewProvider(databaseUrl)
	if err != nil {
		l.LogActivity("Error while establishes a connection with database", err)
		log.Fatalln("Failed to establishes a connection with database", err)
	}

	// schema services
	s := service.NewService(r).
		WithLogHarbour(l).
		WithDatabase(query)

	s.RegisterRoute(http.MethodGet, "/wfschemaget", SchemaGet)
	s.RegisterRoute(http.MethodDelete, "/wfschemadelete", SchemaDelete)
	s.RegisterRoute(http.MethodPost, "/wfschemList", SchemaList)
	s.RegisterRoute(http.MethodPost, "/WFschemaNew", SchemaNew)
	s.RegisterRoute(http.MethodPut, "/WFschemaUpdate", SchemaUpdate)

	return r, nil

}

// TearDown purges the resources and removes the network.
func TearDown(pool *dockertest.Pool, resource *dockertest.Resource) error {
	if err := pool.Purge(resource); err != nil {
		return fmt.Errorf("could not purge resource: %v", err)
	}
	return nil
}

func ternMigrate(databaseUrl string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, databaseUrl)
	if err != nil {
		log.Fatalln("unable to connect", err)
	}

	// Create a new Tern migration instance
	m, err := migrate.NewMigrator(ctx, conn, versionTable)
	if err != nil {
		log.Fatal("Error creating migration instance:", err)
	}
	if err := m.LoadMigrations("./../../db/migrations/"); err != nil {
		log.Fatal("Error loading data:", err)
	}
	if err = m.Migrate(ctx); err != nil {
		log.Fatal("Error loading data:", err)
	}
}