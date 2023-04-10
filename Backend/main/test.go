package main

import (
	"context"
	"database/sql"
	"log"
	"time"
)

type connection struct {
	host     string
	port     string
	con_type string
}

var DB_master = connection{"10.0.0.8", "5506", "tcp"}

func main() {
	db, err := sql.Open("mysql", "root:password"+"@tcp("+DB_master.host+":"+DB_master.port+")/mydb")
	if err != nil {
		log.Printf("Error connecting to database")
	}
	main2(db)
}
func main2(db *sql.DB) {
	//var questions [10]int
	//var code string
	query := "show databases"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, _ := db.PrepareContext(ctx, query)

	defer stmt.Close()
	row := stmt.QueryRowContext(ctx)
	log.Print(row)
	//err = row.Scan(&code, &questions[0], &questions[1], &questions[2], &questions[3], &questions[4], &questions[5], &questions[6], &questions[7], &questions[8], &questions[9])

}
