package rethinkdb

import (
		r "github.com/dancannon/gorethink"
)

// Create connection to rethinkdb 
func createConnection(dbaddr string, debug bool) {
	var err error
	if os.Getenv("DB_PORT_28015_TCP_ADDR") != "" {
		session, err = r.Connect(r.ConnectOpts{
			Address:  os.Getenv("DB_PORT_28015_TCP_ADDR") + ":" + "28015",
			Database: "enforcer_db",
			})
		if debug {
			log.Println("Connecting to database via linked container:", os.Getenv("DB_PORT_28015_TCP_ADDR")+":"+"28015")
		}
	} else {
		session, err = r.Connect(r.ConnectOpts{
			Address:  dbaddr,
			Database: "enforcer_db",
			})
		if debug {
			log.Println("Connecting to datbase:", dbaddr)
		}
	}

	if err != nil {
		log.Fatalln(err.Error())
	}
}


func createDatabase(database string, debug bool) {
	// create database
	_, err := r.DbCreate(database).Run(session)
	if err != nil {
		if debug {
			log.Println(err.Error())
		}
	}
}


func createTable(tables []string, debug bool) {
	for _, table := range tables {
		response, err := r.Db("enforcer_db").TableCreate(table).Run(session)
		if err != nil {
			if debug {
				log.Println(err.Error())
			}
		} else {
			if debug {
				log.Println(response)
			}
		}
	}

}
