package main

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"github.com/go-sql-driver/mysql"
)

var createTableStatements = []string{
	`CREATE DATABASE IF NOT EXISTS library DEFAULT CHARACTER SET = 'utf8' DEFAULT COLLATE 'utf8_general_cli';`,
	`USE library`,
	`CREATE TABLE IF NOT EXISTS novels(
		id INT UNSIGNED NOT NULL AUTO_INCREMENT,
		title VARCHAR(255) NULL,
		author VARCHAR(255) NULL,
		publishedDate VARCHAR(255) NULL,
		imageUrl VARCHAR(255) NULL,
		description TEXT NULL,
		createdBy VARCHAR(255) NULL,
		createdById VARCHAR(255) NULL,
		PRIMARY KEY (id)
	)`,
}

type mysqlDB struct {
	conn   *sql.DB
	list   *sql.Stmt
	listBy *sql.Stmt
	insert *sql.Stmt
	get    *sql.Stmt
	update *sql.Stmt
	delete *sql.Stmt
}

var _ NovelDatabase = &mysqlDB{}

type MySQLConfig struct {
	Username   string
	Password   string
	Host       string
	Port       int
	UnixSocket string
}

func (c MySQLConfig) dataStoreName(databaseName string) string {
	var cred string
	if c.Username != "" {
		cred = c.Username
		if c.Password != "" {
			cred = cred + ":" + c.Password
		}
		cred = cred + "@"
	}
	if c.UnixSocket != "" {
		return fmt.Sprintf("%sunix(%s)/%s", cred, c.UnixSocket, databaseName)
	}
	return fmt.Sprintf("%stcp[%s]:%d/%s", cred, c.Host, c.Port, databaseName)
}

func newMySQLDB(config MySQLConfig) (NovelDatabase, error) {
	if err := config.ensureTableExists(); err != nil {
		return nil, err
	}
	conn, err := sql.Open("mysql", config.dataStoreName("library"))
	if err != nil {
		return nil, fmt.Errorf("mysql: could not get a connection: %v", err)
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("mysql: could not establish a good connection: %v", err)
	}
	db := &mysqlDB{
		conn: conn,
	}

	if db.list, err = conn.Prepare(listStatement); err != nil {
		return nil, fmt.Errorf("mysql: prepare list: %v", err)
	}
}

func (db *mysqlDB) Close() {
	db.conn.Close()
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanBook(s rowScanner) (*Novel, error) {
	// todo
}

const listStatement = `SELECT * FROM novels ORDER BY title`

func (db *mysqlDB) ListBooks() ([]*Novel, error) {
	rows, err := db.list.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var novel Novel
	for rows.Next() {
		novel, err := scanBook(rows)
	}
}

func (db *mysqlDB) ListNovels() ([]*Novel, error) {
	panic("implement me")
}

func (db *mysqlDB) ListNovelsCreatedBy(userID string) ([]*Novel, error) {
	panic("implement me")
}

func (db *mysqlDB) GetNovel(id int64) (*Novel, error) {
	panic("implement me")
}

func (db *mysqlDB) AddNovel(b *Novel) (id int64, err error) {
	panic("implement me")
}

func (db *mysqlDB) DeleteNovel(id int64) error {
	panic("implement me")
}

func (db *mysqlDB) UpdateBook(b *Novel) error {
	panic("implement me")
}

func (config MySQLConfig) ensureTableExists() error {
	conn, err := sql.Open("mysql", config.dataStoreName(""))
	if err != nil {
		return fmt.Errorf("mysql: could not get a connection: %v", err)
	}
	defer conn.Close()

	if conn.Ping() == driver.ErrBadConn {
		return fmt.Errorf("mysql: could not connect to the database." +
			"could be bad address, or this address is not whiteisted for access")
	}

	if _, err := conn.Exec("USE library"); err != nil {
		if mErr, ok := err.(*mysql.MySQLError); ok && mErr.Number == 1049 {
			return createTable(conn)
		}
	}

	if _, err := conn.Exec("DESCRIBE novels"); err != nil {
		if mErr, ok := err.(*mysql.MySQLError); ok && mErr.Number == 1146 {
			return createTable(conn)
		}
		return fmt.Errorf("mysql: could not connect to the database: %v", err)
	}
	return nil
}

func createTable(conn *sql.DB) error {
	for _, stmt := range createTableStatements {
		_, err := conn.Exec(stmt)
		if err != nil {
			return err
		}
	}
	return nil
}
