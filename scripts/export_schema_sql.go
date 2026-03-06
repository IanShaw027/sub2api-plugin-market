// export_schema_sql exports the current Ent schema as PostgreSQL DDL.
// Run with: go run scripts/export_schema_sql.go [-o migrations/000001_initial_schema.up.sql]
//
// Requires DB_* env vars (or defaults) for dialect. The database does not need to exist;
// the driver is used only for dialect detection.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	_ "github.com/lib/pq"
)

func main() {
	out := flag.String("o", "migrations/000001_initial_schema.up.sql", "output file path")
	flag.Parse()

	dsn := buildDSN()
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open ent client: %v", err)
	}
	defer client.Close()

	f, err := os.Create(*out)
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer f.Close()

	if err := client.Schema.WriteTo(context.Background(), f); err != nil {
		log.Fatalf("failed to write schema: %v", err)
	}
	log.Printf("Schema exported to %s", *out)
}

func buildDSN() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5433")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "plugin_market")
	ssl := getEnv("DB_SSLMODE", "disable")
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, ssl)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
