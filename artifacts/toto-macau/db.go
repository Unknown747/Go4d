package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite", "./toto.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	createTables()
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tanggal TEXT NOT NULL,
			sesi INTEGER NOT NULL,
			nomor TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tanggal, sesi)
		)`,
		`CREATE TABLE IF NOT EXISTS predictions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tanggal TEXT NOT NULL,
			sesi INTEGER NOT NULL,
			metode TEXT NOT NULL,
			nomor_list TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			log.Fatal("Error creating table:", err)
		}
	}
}

type Result struct {
	ID        int    `json:"id"`
	Tanggal   string `json:"tanggal"`
	Sesi      int    `json:"sesi"`
	Nomor     string `json:"nomor"`
	CreatedAt string `json:"created_at"`
}

type Prediction struct {
	ID        int    `json:"id"`
	Tanggal   string `json:"tanggal"`
	Sesi      int    `json:"sesi"`
	Metode    string `json:"metode"`
	NomorList string `json:"nomor_list"`
	CreatedAt string `json:"created_at"`
}

func saveResult(tanggal string, sesi int, nomor string) error {
	_, err := db.Exec(
		`INSERT OR REPLACE INTO results (tanggal, sesi, nomor) VALUES (?, ?, ?)`,
		tanggal, sesi, nomor,
	)
	return err
}

func savePredictions(tanggal string, sesi int, metode string, nomorList []string) error {
	joined := joinStrings(nomorList, ",")
	_, err := db.Exec(
		`INSERT INTO predictions (tanggal, sesi, metode, nomor_list) VALUES (?, ?, ?, ?)`,
		tanggal, sesi, metode, joined,
	)
	return err
}

func getRecentResults(limit int) []Result {
	rows, err := db.Query(
		`SELECT id, tanggal, sesi, nomor, created_at FROM results ORDER BY tanggal DESC, sesi DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var r Result
		rows.Scan(&r.ID, &r.Tanggal, &r.Sesi, &r.Nomor, &r.CreatedAt)
		results = append(results, r)
	}
	return results
}

func getLatestPredictions(tanggal string, sesi int) []Prediction {
	rows, err := db.Query(
		`SELECT id, tanggal, sesi, metode, nomor_list, created_at FROM predictions WHERE tanggal = ? AND sesi = ? ORDER BY created_at DESC`,
		tanggal, sesi,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var preds []Prediction
	seen := map[string]bool{}
	for rows.Next() {
		var p Prediction
		rows.Scan(&p.ID, &p.Tanggal, &p.Sesi, &p.Metode, &p.NomorList, &p.CreatedAt)
		key := fmt.Sprintf("%s_%d_%s", p.Tanggal, p.Sesi, p.Metode)
		if !seen[key] {
			seen[key] = true
			preds = append(preds, p)
		}
	}
	return preds
}

func getTodayResult(tanggal string, sesi int) (Result, bool) {
	var r Result
	err := db.QueryRow(
		`SELECT id, tanggal, sesi, nomor, created_at FROM results WHERE tanggal = ? AND sesi = ?`,
		tanggal, sesi,
	).Scan(&r.ID, &r.Tanggal, &r.Sesi, &r.Nomor, &r.CreatedAt)
	if err != nil {
		return r, false
	}
	return r, true
}

func getAllHistory(limit int) []map[string]interface{} {
	rows, err := db.Query(`
		SELECT r.tanggal, r.sesi, r.nomor, 
		       COALESCE(GROUP_CONCAT(p.nomor_list, '||'), '') as predictions
		FROM results r
		LEFT JOIN predictions p ON r.tanggal = p.tanggal AND r.sesi = p.sesi AND p.metode = 'GABUNGAN'
		GROUP BY r.tanggal, r.sesi, r.nomor
		ORDER BY r.tanggal DESC, r.sesi DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var tanggal, nomor, preds string
		var sesi int
		rows.Scan(&tanggal, &sesi, &nomor, &preds)

		predList := ""
		if preds != "" {
			parts := splitString(preds, "||")
			if len(parts) > 0 {
				predList = parts[0]
			}
		}

		history = append(history, map[string]interface{}{
			"tanggal":     tanggal,
			"sesi":        sesi,
			"nomor":       nomor,
			"predictions": predList,
		})
	}
	return history
}

func todayStr() string {
	return time.Now().Format("2006-01-02")
}

func nextSessionInfo() (string, int) {
	today := todayStr()

	_, s1ok := getTodayResult(today, 1)
	if !s1ok {
		return today, 1
	}
	_, s2ok := getTodayResult(today, 2)
	if !s2ok {
		return today, 2
	}

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	return tomorrow, 1
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	result := []string{}
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}
