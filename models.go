package main

import (
	"time"
)

type Machine struct {
	Id          string `gorethink:"id,omitempty"`
	Description string
	Mgm         string
	Ip          string
	Gateway     string
	Dns         string
	Netmask     string
	Append      string
	Initrd      string
	Kernel      string
	Mirror      string
	Owner       string
	Script      string
	Template    string
	Version     string
	Status      string
	Tags        string
	Created     time.Time
}

func (t *Machine) Installed() bool {
	return t.Status == "INSTALLED"
}

func NewMachine(
	id string,
	description string,
	mgm string,
	ip string,
	gateway string,
	dns string,
	netmask string,
	append string,
	initrd string,
	kernel string,
	mirror string,
	owner string,
	script string,
	template string,
	version string,
	status string,
	tags string,
) *Machine {
	return &Machine{
		Id:          id,
		Description: description,
		Mgm:         mgm,
		Ip:          ip,
		Gateway:     gateway,
		Dns:         dns,
		Netmask:     netmask,
		Append:      append,
		Initrd:      initrd,
		Kernel:      kernel,
		Mirror:      mirror,
		Owner:       owner,
		Script:      script,
		Template:    template,
		Version:     version,
		Status:      "UNAVAILABLE",
		Tags:        tags,
	}
}

type Agent struct {
	Agent    string `gorethink:"id,omitempty"`
	Addr    string
	Tags    string
	Status  string
	Created     time.Time
}

func NewAgent(
	id string,
	addr string,
	tags string,
) *Agent {
	return &Agent{
		Agent: id,
		Addr:    addr,
		Tags:    tags,
		Status:  "UNAVAILABLE",
	}
}
