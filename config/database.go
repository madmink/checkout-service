package config

import (
	"checkout-service/log"
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func InitDBConnection(cfg *ApplicationConfig) (map[string]*sql.DB, error) {
	dbConfig := make(map[string]*sql.DB)
	for dbName, db := range cfg.Database {
		db.Password = cfg.Database["checkout_database"].Password
		dbCon, err := DbConnection(db)
		if err != nil {
			return nil, err
		}

		dbConfig[dbName] = dbCon
	}

	return dbConfig, nil
}

func DbConnection(cfg DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn(cfg))
	if err != nil {
		log.Logging.Error.Errorf("Error %s when opening DB\n", err)
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Minute * cfg.ConnMaxLifetime)

	ctx, cancelFunc := context.WithTimeout(context.Background(), cfg.Timeout*time.Second)
	defer cancelFunc()
	err = db.PingContext(ctx)
	if err != nil {
		log.Logging.Error.Errorf("Errors %s pinging DB", err)
		return nil, err
	}
	log.Logging.Access.Infof("Connected to DB %s successfully", cfg.DBname)
	return db, nil
}

func dsn(cfg DatabaseConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s/%s",
		url.QueryEscape(cfg.Username), url.QueryEscape(cfg.Password),
		cfg.Hostname, cfg.DBname)
}
