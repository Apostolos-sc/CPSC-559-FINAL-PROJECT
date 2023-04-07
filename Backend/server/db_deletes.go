package server

import (
	"database/sql"
	"log"
)

func deleteGameRoom(db *sql.DB, accessCode string) error {
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

func deleteRoomUser(db *sql.DB, username string) error {
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

func deleteRoomQuestions(db *sql.DB, accessCode string) error {
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
