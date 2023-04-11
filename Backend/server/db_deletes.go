package main

import (
	"context"
	"database/sql"
	"log"
	"time"
)
func ping(db1 *sql.DB, db2 *sql.DB) *sql.DB {
    //Ping first db
    err := db1.Ping()
    if err != nil {
    	log.Printf("There was an issue when pinging db1.")
    	//Ping second db
    	err = db2.Ping()
    	if err != nil {
    	    log.Printf("There was an issue when pinging db2.")
    	}
    	//we assume db2 will be up if first failed
    	return db2
    } else {
        return db1
    }
}

func deleteGameRoom(db1 *sql.DB, db2 *sql.DB,accessCode string) error {
    db := ping(db1, db2)
	query := "DELETE FROM gameRoom WHERE accessCode = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err.Error())
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, accessCode)
	if err != nil {
		log.Printf("Error %s when deleting gameRoom %s.\n", err.Error(), accessCode)
		return err
	}
	_, err = res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected.\n", err.Error())
		return err
	}
	log.Printf("Game Room %s was successfully deleted.", accessCode)
	return nil
}

func deleteRoomUser(db1 *sql.DB, db2 *sql.DB,username string) error {
    db := ping(db1, db2)
	query := "DELETE FROM roomUser WHERE username = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err.Error())
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, username)
	if err != nil {
		log.Printf("Error %s when deleting row from roomUser table", err.Error())
		return err
	}
	_, err = res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err.Error())
		return err
	}
	log.Printf("Room User %s was successfully deleted.", username)
	return nil
}

func deleteRoomUsers(db1 *sql.DB, db2 *sql.DB, accessCode string) error {
    db := ping(db1, db2)
	query := "DELETE FROM roomUser WHERE accessCode = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err.Error())
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, accessCode)
	if err != nil {
		log.Printf("Error %s when deleting all rows of roomUser in gameRoom %s", err.Error(), accessCode)
		return err
	}
	_, err = res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err.Error())
		return err
	}
	log.Printf(" All Room Users of room %s was successfully deleted.", accessCode)
	return nil
}

func deleteRoomQuestions(db1 *sql.DB, db2 *sql.DB, accessCode string) error {
    db := ping(db1, db2)
	query := "DELETE FROM roomQuestions WHERE accessCode = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err.Error())
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, accessCode)
	if err != nil {
		log.Printf("Error %s when deleting gameRoom %s.\n", err.Error(), accessCode)
		return err
	}
	_, err = res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected.\n", err.Error())
		return err
	}
	log.Printf("Game Room Questions %s were successfully deleted.", accessCode)
	return nil
}
