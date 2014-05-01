package main

import (
	// "flag"
	flag "github.com/dotcloud/docker/pkg/mflag"
	"log"
)

func main() {
	var (
		addr, dbaddr          string
		tls, daemon, debug, h bool
		tables                []string
	)
	
	flag.StringVar(&addr, []string{"addr", "-addr"}, "localhost:4321", "Defaults to 127.0.0.1:4321")
	flag.StringVar(&dbaddr, []string{"dbaddr", "-db"}, "localhost:28015", "Defaults to localhost:28015")
	flag.BoolVar(&daemon, []string{"d"}, false, "Tells enforcer to be act as a web app")
	flag.BoolVar(&debug, []string{"debug"}, false, "Enable debugging")
	flag.BoolVar(&tls, []string{"tls"}, false, "use tls")
	flag.BoolVar(&h, []string{"h", "#help", "-help"}, false, "display the help")
	flag.Parse()

	server := NewServer(addr)
	if h {
		flag.PrintDefaults()
	} else if daemon {

		createConnection(dbaddr, debug)
		database := "enforcer_db"
		createDatabase(database, debug)

		tables = append(tables, "ipxe")
		tables = append(tables, "agents")
		tables = append(tables, "network12")
		createTable(tables, debug)

		if tls {
			StartTlsServer(server)
			if debug {
				log.Println("Started tls-server:", addr)
			}
		} else {
			StartServer(server)
			if debug {
				log.Println("Starting server:", addr)
			}

		}

	} else {
		flag.PrintDefaults()
	}

}
