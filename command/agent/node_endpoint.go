package agent

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

)

// Returns a complete list of all nodes
func (s *HTTPServer) Nodes(w http.ResponseWriter, req *http.Request) {
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

func (s *HTTPServer) ToggleNode(w http.ResponseWriter, req *http.Request) {
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

func (s *HTTPServer) DeleteNode(w http.ResponseWriter, req *http.Request) {
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

func (s *HTTPServer) NodeIpxe(w http.ResponseWriter, req *http.Request) {
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
