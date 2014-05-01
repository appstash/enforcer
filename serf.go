package main

import (
	"encoding/json"
	"flag"
	"fmt"
	r "github.com/dancannon/gorethink"
	"github.com/hashicorp/serf/client"
//	"github.com/ryanuber/columnize"
	"log"
	"net"
	"strings"
)

var (
	debug        bool
	tableMembers = "network12"
)

// RPCAddrFlag returns a pointer to a string that will be populated
// when the given flagset is parsed with the RPC address of the Serf.
func RPCAddrFlag(f *flag.FlagSet) *string {
	return f.String("rpc-addr", "127.0.0.1:7373",
		"RPC address of the Serf agent")
}

// RPCClient returns a new Serf RPC client with the given address.
func RPCClient(addr string) (*client.RPCClient, error) {
	return client.NewRPCClient(addr)
}

// Format some raw data for output. For better or worse, this currently forces
// the passed data object to implement fmt.Stringer, since it's pretty hard to
// implement a canonical *-to-string function.
func formatOutput(data interface{}) ([]byte, error) {
	var out string

	jsonout, err := json.MarshalIndent(data.(fmt.Stringer), "", "  ")
	if err != nil {
		return nil, err
	}
	out = string(jsonout)

	return []byte(prepareOutput(out)), nil
}

// Apply some final formatting to make sure we don't end up with extra newlines
func prepareOutput(in string) string {
	return strings.TrimSpace(string(in))
}

// A container of member details. Maintaining a command-specific struct here
// makes sense so that the agent.Member struct can evolve without changing the
// keys in the output interface.
type Member struct {
	Name   string            `json:"name"`
	Addr   string            `json:"addr"`
	Port   uint16            `json:"port"`
	Tags   map[string]string `json:"tags"`
	Status string            `json:"status"`
	Proto  map[string]uint8  `json:"protocol"`
}

func listMembers(
	Name string,
	Addr string,
	Status string,
) *Member {
	return &Member{
		Name:   Name,
		Addr:   Addr,
		Status: Status,
	}
}

type MemberContainer struct {
	Members []Member `json:"members"`
}

/*func (c MemberContainer) String() string {
	var result []string
	for _, member := range c.Members {
		// Format the tags as tag1=v1,tag2=v2,...
		var tagPairs []string
		for name, value := range member.Tags {
			tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", name, value))
		}
		tags := strings.Join(tagPairs, ",")

		line := fmt.Sprintf("%s|%s|%s|%s",
			member.Name, member.Addr, member.Status, tags)

		result = append(result, line)
	}
	output, _ := columnize.SimpleFormat(result)
	return output
}
*/
func getAllMembers() {

	addr := "127.0.0.1:7373"
	client, err := RPCClient(addr)
	//client, err := RPCClient(*rpcAddr)
	if err != nil {
		fmt.Sprintf("Error connecting to Serf agent: %s", err)
		return
	}
	defer client.Close()

	members, err := client.Members()
	if err != nil {
		if debug {
			log.Println("Error retrieving members: %s", err)
		}
		return
	}
	//fmt.Println(members)

	result := MemberContainer{}

	for _, member := range members {

		addr := net.TCPAddr{IP: member.Addr, Port: int(member.Port)}

		result.Members = append(result.Members, Member{
			Name:   member.Name,
			Addr:   addr.String(),
			Port:   member.Port,
			Tags:   member.Tags,
			Status: member.Status,
			Proto: map[string]uint8{
				"min":     member.DelegateMin,
				"max":     member.DelegateMax,
				"version": member.DelegateCur,
			},
		})
	}
	r.Db("enforcer_db").TableDrop(tableMembers).Run(session)
	var tables []string
	tables = append(tables, tableMembers)
	createTable(tables, debug)

	// Insert the new item into the database
	_, err = r.Table(tableMembers).Insert(result.Members).RunWrite(session)
	if err != nil {
		if debug {
			log.Println("Error inserting data in rethindkdb", err)
		}
		return
	}
}
