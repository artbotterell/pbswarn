package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3" // driver for sql
)

var db *sql.DB
var err error
var sqlStmt string

func main() {
	InitDBs()
	fmt.Println("done")
}

// InitDBs ...
func InitDBs() {

	os.Remove("/home/pbs/warn.db")

	db, err = sql.Open("sqlite3", "/home/pbs/warn.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// create alerts table
	sqlStmt = `
	create table alerts (id integer not null primary key, identifier text, sender text, sent string, status text, msgType text, source text, code text, note text, replacedBy integer);
	delete from alerts;
	`
	execute(sqlStmt)

	// Create infos table
	sqlStmt = `
	create table infos (id integer not null primary key, alertId integer, language text, category text, event text, responseType text, urgency text, severity text, certainty text, expires string, senderName, text, slug text, effective string, onset string, audience text, headline text, cmam text, description text, instruction text, contact text, web text);
	delete from infos;
	`
	execute(sqlStmt)

	// create eventCodes table
	sqlStmt = `
	create table eventCodes (id integer not null primary key, infoId integer, valueName text, value text);
	delete from eventCodes;
	`
	execute(sqlStmt)

	// create parameters table
	sqlStmt = `
	create table parameters (id integer not null primary key, infoId integer, valueName text, value text);
	delete from parameters;
	`
	execute(sqlStmt)

	// create resources table
	sqlStmt = `
	create table resources (id integer not null primary key, infoId integer, resourceDesc text, mimeType text, size integer, uri text, defefUri text, digest text);
	delete from resources;
	`
	execute(sqlStmt)

	// create areas table
	sqlStmt = `
	create table areas (id integer not null primary key, infoId integer, areaDesc text, altitude int, ceiling integer);
	delete from areas;
	`
	execute(sqlStmt)

	// create geocodes table
	sqlStmt = `
	create table geocodes (id integer not null primary key, areaId integer, valueName text, value text);
	delete from geocodes;
	`
	execute(sqlStmt)

	// create polygons table
	sqlStmt = `
	create table polygons (id integer not null primary key, areaId integer, polygon text);
	delete from polygons;
	`
	execute(sqlStmt)

	// create circles table
	sqlStmt = `
	create table circles (id integer not null primary key, areaId integer, circle text);
	delete from circles;
	`
	execute(sqlStmt)

	// create CAP table
	sqlStmt = `
	create table CAP (id integer not null primary key, alertId integer, xml text);
	delete from CAP;
	`
	execute(sqlStmt)

}

func execute(sqlStmt string) {
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Printf("Error in execute(): %q: %s\n", err, sqlStmt)
		return
	}
}

// Query ...
func Query(sqlStmt string) *sql.Rows {
	rows, err := db.Query(sqlStmt)
	defer rows.Close()
	if err != nil {
		log.Printf("Error in query():  %q: %s\n", err, sqlStmt)
	}
	return rows
}
