package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4" // PostgrerSQL driver
	"github.com/sdslabs/gasper/configs"
	"github.com/sdslabs/gasper/types"
)

var (
	postgresqlHost         = `%`
	postgresqlRootUser     = configs.ServiceConfig.DbMaker.PostgreSQL.Env["POSTGRES_USER"]
	postgresqlPassword     = configs.ServiceConfig.DbMaker.PostgreSQL.Env["POSTGRES_PASSWORD"]
	postgresqlDatabaseName = configs.ServiceConfig.DbMaker.PostgreSQL.Env["POSTGRES_DB"]
	postgresqlPort         = configs.ServiceConfig.DbMaker.PostgreSQL.ContainerPort
)

// CreatePostgresqlDB creates a postgre database
func CreatePostgresqlDB(db types.Database) error {
	ctx := context.Background()
	connection := fmt.Sprintf("postgres://%v:%v@localhost:%d/%v", postgresqlRootUser, postgresqlPassword, postgresqlPort, postgresqlRootUser)
	conn, err := pgx.Connect(ctx, connection)
	if err != nil {
		return fmt.Errorf("Error while creating the database : %s", err)
	}
	defer conn.Close(ctx)

	if _, err = conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", db.GetName())); err != nil {
		return fmt.Errorf("Error while creating the database : Database Already Exists")
	}

	query := fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", db.GetUser(), db.GetPassword())
	if _, err = conn.Exec(ctx, query); err != nil {
		if err = refreshPostgresqlUser(db, conn); err != nil {
			return fmt.Errorf("Error while creating the database : %s", err)
		}
	}

	query = fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO %s", db.GetName(), db.GetUser())
	if _, err = conn.Exec(ctx, query); err != nil {
		return fmt.Errorf("Error while creating the database : %s", err)
	}

	query = fmt.Sprintf("REVOKE CONNECT ON DATABASE %s FROM PUBLIC;", db.GetName())
	if _, err = conn.Exec(ctx, query); err != nil {
		return fmt.Errorf("Error while revoking permissions : %s", err)
	}
	return nil
}

// DeletePostgresqlDB deletes the database given by the database name and username
func DeletePostgresqlDB(databaseName string) error {
	username := databaseName
	ctx := context.Background()

	connection := fmt.Sprintf("postgres://%v:%v@localhost:%d/%v", postgresqlRootUser, postgresqlPassword, postgresqlPort, postgresqlRootUser)
	conn, err := pgx.Connect(ctx, connection)
	if err != nil {
		return fmt.Errorf("Error while creating the database : %s", err)
	}
	defer conn.Close(ctx)

	if _, err = conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName)); err != nil {
		return fmt.Errorf("Error while deleting the database : %s", err)
	}

	if _, err = conn.Exec(ctx, fmt.Sprintf("DROP USER IF EXISTS %s", username)); err != nil {
		return fmt.Errorf("Error while deleting the user : %s", err)
	}
	return nil
}

func refreshPostgresqlUser(db types.Database, conn *pgx.Conn) error {
	ctx := context.Background()
	_, err := conn.Exec(ctx, fmt.Sprintf("DROP USER IF EXISTS '%s'@'%s'", db.GetUser(), postgresqlHost))
	if err != nil {
		return fmt.Errorf("Error while deleting the user : %s", err)
	}

	query := fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", db.GetUser(), db.GetPassword())
	if _, err = conn.Exec(ctx, query); err != nil {
		return fmt.Errorf("Error while creating the database : %s", err)
	}
	return nil
}
