package main

import (
	"fmt"
	r "github.com/dancannon/gorethink"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	router      *mux.Router
	session     *r.Session
	tableIpxe   = "ipxe"
	tableAgents = "agents"
	agentId     = "network12"
	agentAddr   = "127.0.0.1:7373"
	tags        = "nothing"
)

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
func NewServer(addr string) *http.Server {
	// Setup router
	router = initRouting()

	// Create and start server
	return &http.Server{
		Addr:    addr,
		Handler: router,
	}
}

func StartTlsServer(server *http.Server) {
	err := server.ListenAndServeTLS("cert.pem", "key.pem")
	if err != nil {
		log.Fatalln("Error: %v", err)
	}
}

func StartServer(server *http.Server) {
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalln("Error: %v", err)
	}
}

func initRouting() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/active", activeIndexHandler)
	r.HandleFunc("/completed", completedIndexHandler)
	r.HandleFunc("/newnode", newNodeHandler)
	r.HandleFunc("/cluster", clusterHandler)
	r.HandleFunc("/cluster", clusterHandler)
	r.HandleFunc("/add/agent", newNewAgentHandler)
	r.HandleFunc("/view/{id}", viewHandler)
	r.HandleFunc("/get/members/{id}", getMemberHandler)
	r.HandleFunc("/members/{id}", membersHandler)
	r.HandleFunc("/ipxe/{id}", executeHandler)
	r.HandleFunc("/agent/{id}", executeAgentHandler)
	r.HandleFunc("/toggle/{id}", toggleHandler)
	r.HandleFunc("/toggleagent/{id}", toggleAgentHandler)
	r.HandleFunc("/delete/{id}", deleteHandler)
	r.HandleFunc("/deleteagent/{id}", deleteAgentHandler)
	r.HandleFunc("/clear", clearHandler)

	// Add handler for static files
	
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("static")))
	return r
}

// Handlers

func indexHandler(w http.ResponseWriter, req *http.Request) {
	instructions := []Machine{}

	// Fetch all the items from the database
	rows, err := r.Table(tableIpxe).OrderBy(r.Asc("Created")).Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Scan each row into a TodoItem instance and then add this to the list
	for rows.Next() {
		var instruction Machine

		err := rows.Scan(&instruction)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		instructions = append(instructions, instruction)
	}

	renderHtml(w, "enforcer", instructions)
}

func getMemberHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	if id == "" {
		http.NotFound(w, req)
		return
	}

	// Check that the item exists
	res, err := r.Table(tableAgents).Get(id).RunRow(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res.IsNil() {
		http.NotFound(w, req)
		return
	}

	getAllMembers()

    http.Redirect(w, req, "/members/" + id , http.StatusFound)
}

func membersHandler(w http.ResponseWriter, req *http.Request) {
	members := []Member{}

	// Fetch all the items from the database
	rows, err := r.Table(tableMembers).Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Scan each row into a TodoItem instance and then add this to the list
	for rows.Next() {
		var addMember Member

		err := rows.Scan(&addMember)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		members = append(members, addMember)
	}

	renderHtml(w, "members", members)
}

func clusterHandler(w http.ResponseWriter, req *http.Request) {
	agents := []Agent{}

	// Fetch all the items from the database
	rows, err := r.Table(tableAgents).Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Scan each row into a TodoItem instance and then add this to the list
	for rows.Next() {
		var addAgent Agent

		err := rows.Scan(&addAgent)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		agents = append(agents, addAgent)
	}

	renderHtml(w, "cluster", agents)
}

func activeIndexHandler(w http.ResponseWriter, req *http.Request) {
	instructions := []Machine{}

	// Fetch all the items from the database
	query := r.Table(tableIpxe).Filter(r.Row.Field("Status").Eq("AVAILABLE"))
	query = query.OrderBy(r.Asc("Created"))
	rows, err := query.Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	// Scan each row into a TodoItem instance and then add this to the list
	for rows.Next() {
		var instruction Machine

		err := rows.Scan(&instruction)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		instructions = append(instructions, instruction)
	}

	renderTemplate(w, "enforcer", instructions)
}

func completedIndexHandler(w http.ResponseWriter, req *http.Request) {
	instructions := []Machine{}

	// Fetch all the items from the database
	query := r.Table(tableIpxe).Filter(r.Row.Field("Status").Eq("INSTALLED"))
	query = query.OrderBy(r.Asc("Created"))
	rows, err := query.Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Scan each row into a TodoItem instance and then add this to the list
	for rows.Next() {
		var instruction Machine

		err := rows.Scan(&instruction)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		instructions = append(instructions, instruction)
	}

	renderTemplate(w, "enforcer", instructions)
}

func newNodeHandler(w http.ResponseWriter, req *http.Request) {
	// Create the item
	ipxe := NewMachine(
		req.PostFormValue("id"),
		req.PostFormValue("description"),
		req.PostFormValue("mgm"),
		req.PostFormValue("ip"),
		req.PostFormValue("gateway"),
		req.PostFormValue("dns"),
		req.PostFormValue("netmask"),
		req.PostFormValue("append"),
		req.PostFormValue("initrd"),
		req.PostFormValue("kernel"),
		req.PostFormValue("mirror"),
		req.PostFormValue("owner"),
		req.PostFormValue("script"),
		req.PostFormValue("template"),
		req.PostFormValue("version"),
		req.PostFormValue("status"),
		req.PostFormValue("tags"))
	ipxe.Created = time.Now()

	templateName := "ipxe/" + ipxe.Template + ".ipxe"
	if _, err := os.Stat(templateName); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			return
		}
	}

	//Insert the new item into the database
	_, err := r.Table(tableIpxe).Insert(ipxe).RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, req, "/", http.StatusFound)
}

func newNewNewAgentHandler(w http.ResponseWriter, req *http.Request) {

	instructions := ""
	renderHtml(w, "newagent", instructions)
}

func newNewAgentHandler(w http.ResponseWriter, req *http.Request) {
	// Create the item
	agent := NewAgent(
		req.PostFormValue("id"),
		req.PostFormValue("agentAddr"),
		req.PostFormValue("tags"))
	agent.Created = time.Now()

	templateName := "ipxe/agent.ipxe"
	if _, err := os.Stat(templateName); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			return
		}
	}

	//Insert the new item into the database
	_, err := r.Table(tableAgents).Insert(agent).RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, req, "/cluster", http.StatusFound)
}

func newAgentHandler(w http.ResponseWriter, req *http.Request) {

	// Setup default cluster agent and insert it in database

	Agent := NewAgent(agentId, agentAddr, tags)

	templateName := "ipxe/agent.ipxe"
	if _, err := os.Stat(templateName); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			return
		}
	}

	//Insert the new item into the database
	_, err := r.Table(tableAgents).Insert(Agent).RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, req, "/cluster", http.StatusFound)
}

func toggleAgentHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	if id == "" {
		http.NotFound(w, req)
		return
	}

	// Check that the item exists
	res, err := r.Table(tableAgents).Get(id).RunRow(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res.IsNil() {
		http.NotFound(w, req)
		return
	}

	_, err = r.Table(tableAgents).Get(id).Update(map[string]interface{}{"Status": r.Branch(
		r.Row.Field("Status").Eq("AVAILABLE"),
		"UNAVAILABLE",
		"AVAILABLE",
	)}).RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, req, "/cluster", http.StatusFound)
}

func viewHandler(w http.ResponseWriter, req *http.Request) {
	instructions := []Machine{}

	vars := mux.Vars(req)
	id := vars["id"]
	fmt.Println(id)
	if id == "" {
		http.NotFound(w, req)
		return
	}

	// Check that the item exists
	res, err := r.Table(tableIpxe).Get(id).RunRow(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res.IsNil() {
		http.NotFound(w, req)
		return
	}

	// Fetch all the items from the database
	row, err := r.Table(tableIpxe).Get(id).RunRow(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var instruction Machine

	err = row.Scan(&instruction)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	instructions = append(instructions, instruction)

	fmt.Println(instructions)
	renderTemplate(w, "machine", instructions)
}

func executeAgentHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	if id == "" {
		http.NotFound(w, req)
		return
	}

	// Check that the item exists
	queryExistance := r.Table(tableAgents)
	queryExistance = queryExistance.Filter(r.Row.Field("id").Eq(id)).Filter(r.Row.Field("Status").Eq("AVAILABLE"))
	res, err := queryExistance.RunRow(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res.IsNil() {
		http.NotFound(w, req)
		return
	}

	agents := []Agent{}

	var addAgent Agent
	err = res.Scan(&addAgent)
	if err != nil {
		fmt.Println(err)
		return
	}

	agents = append(agents, addAgent)

	// Render template
	renderTemplate(w, "agent", agents)

}

func executeHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	if id == "" {
		http.NotFound(w, req)
		return
	}

	// Check that the item exists
	queryExistance := r.Table(tableIpxe)
	queryExistance = queryExistance.Filter(r.Row.Field("id").Eq(id)).Filter(r.Row.Field("Status").Eq("AVAILABLE"))
	res, err := queryExistance.RunRow(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res.IsNil() {
		http.NotFound(w, req)
		return
	}

	instructions := []Machine{}

	var instruction Machine
	err = res.Scan(&instruction)
	if err != nil {
		fmt.Println(err)
		return
	}

	instructions = append(instructions, instruction)

	// Render template
	renderTemplate(w, instruction.Template, instructions)

	// Set status to installed
	_, err = r.Table(tableIpxe).Get(id).Update(map[string]interface{}{"Status": "INSTALLED"}).RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func toggleHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	if id == "" {
		http.NotFound(w, req)
		return
	}

	// Check that the item exists
	res, err := r.Table(tableIpxe).Get(id).RunRow(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res.IsNil() {
		http.NotFound(w, req)
		return
	}

	_, err = r.Table(tableIpxe).Get(id).Update(map[string]interface{}{"Status": r.Branch(
		r.Row.Field("Status").Eq("AVAILABLE"),
		"UNAVAILABLE",
		"AVAILABLE",
	)}).RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, req, "/", http.StatusFound)
}

func deleteHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	if id == "" {
		http.NotFound(w, req)
		return
	}

	// Check that the item exists
	res, err := r.Table(tableIpxe).Get(id).RunRow(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res.IsNil() {
		http.NotFound(w, req)
		return
	}

	// Delete the item
	_, err = r.Table(tableIpxe).Get(id).Delete().RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, "/", http.StatusFound)
}

func deleteAgentHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	if id == "" {
		http.NotFound(w, req)
		return
	}

	// Check that the item exists
	res, err := r.Table(tableAgents).Get(id).RunRow(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res.IsNil() {
		http.NotFound(w, req)
		return
	}

	// Delete the item
	_, err = r.Table(tableAgents).Get(id).Delete().RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, "/cluster", http.StatusFound)
}

func clearHandler(w http.ResponseWriter, req *http.Request) {
	// Delete all completed items
	_, err := r.Table(tableIpxe).Filter(
		r.Row.Field("Status").Eq("INSTALLED"),
	).Delete().RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, "/", http.StatusFound)
}
