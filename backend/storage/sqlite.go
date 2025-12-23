package storage

import (
	"database/sql"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	DB   *sql.DB
	Once sync.Once
)

func InitDB(path string) {
	Once.Do(func() {
		var err error
		DB, err = sql.Open("sqlite3", path)
		if err != nil {
			log.Fatalf("Failed to open sqlite db: %v", err)
		}

		createTableSQL := `
		CREATE TABLE IF NOT EXISTS players (
			uuid TEXT PRIMARY KEY,
			name TEXT,
			funds INTEGER DEFAULT 0,
			items_count INTEGER DEFAULT 0,
			last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		`
		_, err = DB.Exec(createTableSQL)
		if err != nil {
			log.Fatalf("Failed to create table: %v", err)
		}
		log.Println("SQLite Persistence Initialized.")
	})
}

func SavePlayer(uuid string, name string, funds int, itemsCount int) {
	if DB == nil { return }
	
	query := `
	INSERT INTO players (uuid, name, funds, items_count, last_seen)
	VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(uuid) DO UPDATE SET
		name = excluded.name,
		funds = excluded.funds,
		items_count = excluded.items_count,
		last_seen = CURRENT_TIMESTAMP;
	`
	_, err := DB.Exec(query, uuid, name, funds, itemsCount)
	if err != nil {
		log.Printf("Error saving player %s: %v", uuid, err)
	}
}

func LoadPlayer(uuid string) (int, int) {
	if DB == nil { return 0, 0 }
	
	row := DB.QueryRow("SELECT funds, items_count FROM players WHERE uuid = ?", uuid)
	var funds, itemsCount int
	err := row.Scan(&funds, &itemsCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, 0
		}
		log.Printf("Error loading player %s: %v", uuid, err)
		return 0, 0
	}
	return funds, itemsCount
}
