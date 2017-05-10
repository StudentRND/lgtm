package sqlite

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

func Open(conn string) (*Database, error) {
	d, err := sql.Open("sqlite3", conn)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot connect to mysql with %s", conn)
	}
	if err := d.Ping(); err != nil {
		return nil, errors.Wrapf(err, "cannot make initial database connection to %s", conn)
	}

	if err := setupDB(d); err != nil {
		return nil, err
	}

	return &Database{d}, nil
}

func setupDB(db *sql.DB) error {
	const users = `CREATE TABLE IF NOT EXISTS users(
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				slack_id text,
				oauth_token text);`
	_, err := db.Exec(users)
	if err != nil {
		return err
	}

	const hooks = `CREATE TABLE IF NOT EXISTS hooks(
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				hook_id integer,
				user_id integer);`
	_, err = db.Exec(hooks)
	if err != nil {
		return err
	}

	const pullRequests = `CREATE TABLE IF NOT EXISTS pull_requests(
				id INTEGER PRIMARY KEY AUTOINCREMENT,
			 	pull_request_id integer,
				user_id integer,
				timestamp text);`
	_, err = db.Exec(pullRequests)
	if err != nil {
		return err
	}

	return nil
}

type Database struct {
	db *sql.DB
}

func (d *Database) ReadUserAuth(slackId string) (string, error) {
	const query = `SELECT * FROM users WHERE slack_id=?`
	resp := d.db.QueryRow(query, slackId)

	var id int
	var token string
	err := resp.Scan(&id, &slackId, &token)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (d *Database) CreateUser(slackId, oauthToken string) error {
	const query = `INSERT INTO users(slack_id, oauth_token) VALUES (?, ?)`
	_, err := d.db.Exec(query, slackId, oauthToken)
	if err != nil {
		return err
	}

	return nil
}

// exec calls db.db.Exec with passed arguments and returns the id of the LastInsertId
func exec(db *sql.DB, query string, args ...interface{}) (int64, error) {
	resp, err := db.Exec(query, args...)
	if err != nil {
		return 0, errors.Wrapf(err, "unable to exec query: %v", query)
	}

	id, err := resp.LastInsertId()
	if err != nil {
		return 0, errors.Wrapf(err, "unable to get last id after query: %v", query)
	}

	return id, nil
}
