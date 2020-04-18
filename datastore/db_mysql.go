package datastore

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"github.com/IkezawaYuki/go-novel-shelf/domain"
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

var _ domain.NovelDatabase = &mysqlDB{}

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

func NewMySQLDB(config MySQLConfig) (domain.NovelDatabase, error) {
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
	if db.listBy, err = conn.Prepare(listByStatement); err != nil {
		return nil, fmt.Errorf("mysql: prepare listBy: %v", err)
	}
	if db.get, err = conn.Prepare(getStatement); err != nil {
		return nil, fmt.Errorf("mysql: prepare get: %v", err)
	}
	if db.insert, err = conn.Prepare(insertStatement); err != nil {
		return nil, fmt.Errorf("mysql: prepare insert: %v", err)
	}
	if db.update, err = conn.Prepare(updateStatement); err != nil {
		return nil, fmt.Errorf("mysql: prepare update: %v", err)
	}
	if db.delete, err = conn.Prepare(deleteStatement); err != nil {
		return nil, fmt.Errorf("mysql: prepare delete: %v", err)
	}

	return db, nil
}

func (db *mysqlDB) Close() {
	db.conn.Close()
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanBook(s rowScanner) (*domain.Novel, error) {
	var (
		id            int64
		title         sql.NullString
		author        sql.NullString
		publishedDate sql.NullString
		imageURL      sql.NullString
		description   sql.NullString
		createdBy     sql.NullString
		createdByID   sql.NullString
	)
	if err := s.Scan(&id, &title, &author, &publishedDate, &imageURL, &description, &createdBy, &createdByID); err != nil {
		return nil, err
	}
	novel := &domain.Novel{
		ID:            id,
		Title:         title.String,
		Author:        author.String,
		PublishedDate: publishedDate.String,
		ImageURL:      imageURL.String,
		Description:   description.String,
		CreatedBy:     createdBy.String,
		CreatedByID:   createdByID.String,
	}
	return novel, nil
}

const listStatement = `SELECT * FROM novels ORDER BY title`

func (db *mysqlDB) ListNovels() ([]*domain.Novel, error) {
	rows, err := db.list.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var novels []*domain.Novel
	for rows.Next() {
		novel, err := scanBook(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql: could not read row: %v", err)
		}
		novels = append(novels, novel)
	}
	return novels, nil
}

const listByStatement = `
	SELECT * FROM novels
	WHERE createdById = ? ORDER BY title`

func (db *mysqlDB) ListNovelsCreatedBy(userID string) ([]*domain.Novel, error) {
	if userID == "" {
		return db.ListNovels()
	}
	rows, err := db.list.Query(userID)
	if err != nil {
		return nil, err
	}

	var novels []*domain.Novel
	for rows.Next() {
		novel, err := scanBook(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql: could not read row: %v", err)
		}
		novels = append(novels, novel)
	}
	return novels, nil
}

const getStatement = "SELECT * FROM novels WHERE id = ?"

func (db *mysqlDB) GetNovel(id int64) (*domain.Novel, error) {
	novel, err := scanBook(db.get.QueryRow(id))
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("mysql: could not find novel with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("mysql: could not get novel")
	}
	return novel, nil
}

const insertStatement = `
	INSERT INTO novels (
	title, author, publishedDate, imageUrl, description, createdBy, createdById
	) VALUES (?, ?, ?, ?, ?, ?, ?)`

func (db *mysqlDB) AddNovel(b *domain.Novel) (id int64, err error) {
	r, err := execAffectingOneRow(db.insert, b.Title, b.Author, b.PublishedDate, b.ImageURL, b.Description, b.CreatedBy, b.CreatedByID)
	if err != nil {
		return 0, err
	}
	lastInsertID, err := r.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("mysql: could not get last insert ID: %v", err)
	}
	return lastInsertID, nil
}

const deleteStatement = `DELETE FROM novels WHERE id = ?`

// todo check
func (db *mysqlDB) DeleteNovel(id int64) error {
	if id == 0 {
		return fmt.Errorf("mysql: book with unassigned ID passed into deleteBook")
	}
	_, err := execAffectingOneRow(db.delete, id)
	return err
}

const updateStatement = `	
	UPDATE novels 
	SET title = ?, author = ?, publishedDate = ?, imageUrl = ?, description = ?, createdBy = ?, createdById = ?
	WHERE id = ?`

func (db *mysqlDB) UpdateBook(b *domain.Novel) error {
	if b.ID == 0 {
		return fmt.Errorf("mysql: book with unassigned ID passed into updateBook")
	}
	_, err := execAffectingOneRow(db.delete, b.Title, b.Author, b.PublishedDate,
		b.ImageURL, b.Description, b.CreatedBy, b.CreatedByID, b.ID)
	return err
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

func execAffectingOneRow(stmt *sql.Stmt, args ...interface{}) (sql.Result, error) {
	r, err := stmt.Exec(args...)
	if err != nil {
		return r, fmt.Errorf("mysql: could not execute statement: %v", err)
	}
	rowsAffected, err := r.RowsAffected()
	if err != nil {
		return r, fmt.Errorf("mysql: could not get rows affected: %v", err)
	} else if rowsAffected != 1 {
		return r, fmt.Errorf("mysql: expected 1 row affected, got %d", rowsAffected)
	}
	return r, nil
}
