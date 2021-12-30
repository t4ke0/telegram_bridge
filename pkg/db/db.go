package db

import (
	"database/sql"
	"errors"
	"strconv"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// DBfile
const DBfile string = "data.db"

var (
	ErrConflict error = errors.New("already exist")
	ErrInsert   error = errors.New("cannot insert new row")
)

// InitDB
func InitDB() (err error) {

	db, err := sql.Open("sqlite", DBfile)
	if err != nil {
		err = err
		return
	}

	defer func() {
		err = db.Close()
	}()

	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS subscribers (
	telegram_user_id VARCHAR PRIMARY KEY NOT NULL,
	username VARCHAR NOT NULL,
	token VARCHAR NOT NULL
);

CREATE TABLE IF NOT EXISTS messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	telegram_user_id VARCHAR NOT NULL,
	message VARCHAR
);

CREATE TABLE IF NOT EXISTS lastupdate (
	last_update_id VARCHAR NOT NULL
);
	`)

	return
}

// Subscriber
type Subscriber struct {
	ID       int
	Username string
}

// DB
type DB struct {
	conn *sql.DB
}

// New
func New() (DB, error) {

	db, err := sql.Open("sqlite", DBfile)
	if err != nil {
		return DB{}, err
	}

	return DB{
		conn: db,
	}, nil
}

// Close
func (d DB) Close() error {
	return d.conn.Close()
}

// GetSubscriber
func (d DB) GetSubscriber(token string) (s Subscriber, err error) {
	var id, username string

	err = d.conn.QueryRow(`
SELECT telegram_user_id, username FROM subscribers
WHERE token = $1`, token).Scan(&id, &username)
	if err != nil && err != sql.ErrNoRows {
		return
	}

	s.ID, err = strconv.Atoi(id)
	if err != nil {
		return
	}
	s.Username = username

	return
}

// InsertNewUser
func (d DB) InsertNewUser(userID, username string) (string, error) {

	ok, err := d.IsSubscriberExist(userID)
	if err != nil {
		return "", err
	}

	if ok {
		return "", ErrConflict
	}

	token := uuid.New().String()

	r, err := d.conn.Exec(`
INSERT INTO subscribers(telegram_user_id, username, token) VALUES($1, $2, $3)`,
		userID, username, token)
	if err != nil {
		return "", err
	}

	if n, _ := r.RowsAffected(); n == 0 {
		return "", ErrInsert
	}

	return token, nil
}

// InsertNewMessage
func (d DB) InsertNewMessage(userID, message string) error {

	r, err := d.conn.Exec(`
INSERT INTO messages(telegram_user_id, message) VALUES($1, $2)`,
		userID, message)
	if err != nil {
		return err
	}

	if n, _ := r.RowsAffected(); n == 0 {
		return ErrInsert
	}

	return nil
}

// IsSubscriberExist
func (d DB) IsSubscriberExist(userID string) (bool, error) {

	var user string
	err := d.conn.QueryRow(`
SELECT username FROM subscribers
WHERE telegram_user_id = $1
	`, userID).Scan(&user)

	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	if user == "" {
		return false, nil
	}

	return true, nil
}

// AuthorizeToken
func (d DB) AuthorizeToken(token string) (bool, error) {
	var count int
	if err := d.conn.QueryRow(`
SELECT COUNT(*) FROM subscribers
WHERE  token = $1
	`, token).Scan(&count); err != nil && err != sql.ErrNoRows {
		return false, err
	}

	if count == 0 {
		return false, nil
	}

	return true, nil
}

// GetLastUpdateID
func (d DB) GetLastUpdateID() (int, error) {
	var last string
	if err := d.conn.QueryRow(`SELECT last_update_id FROM lastupdate`).Scan(&last); err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	if last == "" {
		return 0, nil
	}

	n, err := strconv.Atoi(last)
	if err != nil {
		return 0, err
	}

	return n, nil
}

// InsertUpdateID
func (d DB) InsertUpdateID(updateID int) error {
	n, err := d.GetLastUpdateID()
	if err != nil {
		return err
	}
	var query string
	if n == 0 {
		query = "INSERT INTO lastupdate(last_update_id) VALUES($1)"
	} else {
		query = "UPDATE lastupdate set last_update_id = $1"
	}

	if _, err := d.conn.Exec(query, updateID); err != nil {
		return err
	}

	return nil
}
