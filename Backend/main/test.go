// package main
//
// import (
//
//	"context"
//	"database/sql"
//	"log"
//	"time"
//
// )
//
//	type connection struct {
//		host     string
//		port     string
//		con_type string
//	}
//
// var DB_master = connection{"10.0.0.8", "5506", "tcp"}
//
//	func main() {
//		db, err := sql.Open("mysql", "root:password"+"@tcp("+DB_master.host+":"+DB_master.port+")/mydb")
//		if err != nil {
//			log.Printf("Error connecting to database")
//		}
//		main2(db)
//	}
//
//	func main2(db *sql.DB) {
//		//var questions [10]int
//		//var code string
//		query := "show databases"
//		ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
//		defer cancelfunc()
//		stmt, _ := db.PrepareContext(ctx, query)
//
//		defer stmt.Close()
//		row := stmt.QueryRowContext(ctx)
//		log.Print(row)
//		//err = row.Scan(&code, &questions[0], &questions[1], &questions[2], &questions[3], &questions[4], &questions[5], &questions[6], &questions[7], &questions[8], &questions[9])
//
// }
package main

import (
	"log"
	"strconv"
	"time"
)

var allowed_duration = 35

func main() {
	ticker := time.NewTicker(1 * time.Second)
	start := allowed_duration
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				//diff := start.Sub(t)
				//output := time.Time{}.Add(diff)
				//log.Printf(t.String())
				//log.Printf(output.Format("15:03:03"))
				//log.Printf(strconv.Itoa(int(start.Sub(t))))
				//log.Printf("30 minus partially string %s", strconv.Itoa(30-int(start.Sub(t))))
				//for _, value := range gameRooms[command[1]] {
				//	err = value.WriteMessage(1, []byte(t.String()))
				//	//log.Printf("the time is %s",t.String())
				//	log.Printf("the time is %s", string(rune(30-int(start.Sub(t)))))
				//	if err != nil {
				//	} else {
				//		log.Printf("Error sending timer to server from time server")
				//	}
				//}
				start--
				log.Printf(strconv.Itoa(start))
				//err = value.WriteMessage(1, []byte(t.String()))
			case <-done:
				ticker.Stop()
			}
		}
	}()
	time.Sleep(time.Duration(allowed_duration) * time.Second) // Stopping the timer at the end of 31 secs
	ticker.Stop()
	done <- true
}
