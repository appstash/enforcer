package agent

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/consul"
	"github.com/mitchellh/mapstructure"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Ports is used to simplify the configuration by
// providing default ports, and allowing the addresses
// to only be specified once
type PortConfig struct {
	HTTP    int // HTTP API
	RPC     int // CLI RPC
	Server  int // Server internal RPC
}

// Config is the configuration that can be set for an Agent.
// Some of this is configurable as CLI flags, but most must
// be set using a configuration file.
type Config struct {
	// Server controls if this agent acts like a Consul server,
	// or merely as a client. Servers have more state, take part
	// in leader election, etc.
	Server bool `mapstructure:"server"`


	// LogLevel is the level of the logs to putout
	LogLevel string `mapstructure:"log_level"`

	// ClientAddr is used to control the address we bind to for
	// client services (DNS, HTTP, RPC)
	ClientAddr string `mapstructure:"client_addr"`

	// BindAddr is used to control the address we bind to.
	// If not specified, the first private IP we find is used.
	// This controls the address we use for cluster facing
	// services (Gossip, Server RPC)
	BindAddr string `mapstructure:"bind_addr"`

	// AdvertiseAddr is the address we use for advertising our Serf,
	// and Consul RPC IP. If not specified, bind address is used.
	AdvertiseAddr string `mapstructure:"advertise_addr"`

	// Port configurations
	Ports PortConfig

	// LeaveOnTerm controls if Serf does a graceful leave when receiving
	// the TERM signal. Defaults false. This can be changed on reload.
	LeaveOnTerm bool `mapstructure:"leave_on_terminate"`

	// SkipLeaveOnInt controls if Serf skips a graceful leave when receiving
	// the INT signal. Defaults false. This can be changed on reload.
	SkipLeaveOnInt bool `mapstructure:"skip_leave_on_interrupt"`

	// StatsiteAddr is the address of a statsite instance. If provided,
	// metrics will be streamed to that instance.
	StatsiteAddr string `mapstructure:"statsite_addr"`

	// Protocol is the Consul protocol version to use.
	Protocol int `mapstructure:"protocol"`

	// EnableDebug is used to enable various debugging features
	EnableDebug bool `mapstructure:"enable_debug"`

	// VerifyIncoming is used to verify the authenticity of incoming connections.
	// This means that TCP requests are forbidden, only allowing for TLS. TLS connections
	// must match a provided certificate authority. This can be used to force client auth.
	VerifyIncoming bool `mapstructure:"verify_incoming"`

	// VerifyOutgoing is used to verify the authenticity of outgoing connections.
	// This means that TLS requests are used. TLS connections must match a provided
	// certificate authority. This is used to verify authenticity of server nodes.
	VerifyOutgoing bool `mapstructure:"verify_outgoing"`

	// CAFile is a path to a certificate authority file. This is used with VerifyIncoming
	// or VerifyOutgoing to verify the TLS connection.
	CAFile string `mapstructure:"ca_file"`

	// CertFile is used to provide a TLS certificate that is used for serving TLS connections.
	// Must be provided to serve TLS connections.
	CertFile string `mapstructure:"cert_file"`

	// KeyFile is used to provide a TLS key that is used for serving TLS connections.
	// Must be provided to serve TLS connections.
	KeyFile string `mapstructure:"key_file"`

	// UiDir is the directory containing the Web UI resources.
	// If provided, the UI endpoints will be enabled.
	UiDir string `mapstructure:"ui_dir"`

	// PidFile is the file to store our PID in
	PidFile string `mapstructure:"pid_file"`

	// ConsulConfig can either be provided or a default one created
	ConsulConfig *consul.Config `mapstructure:"-"`
}

type dirEnts []os.FileInfo

// DefaultConfig is used to return a sane default configuration
func DefaultConfig() *Config {
	return &Config{
		Server:     false,
		LogLevel:   "INFO",
		ClientAddr: "127.0.0.1",
		BindAddr:   "0.0.0.0",
		Ports: PortConfig{
			HTTP:    8500,
			RPC:     8400,
			Server:  8300,
		},
		Protocol:   consul.ProtocolVersionMax,
	}
}


// ClientListener is used to format a listener for a
// port on a ClientAddr
func (c *Config) ClientListener(port int) (*net.TCPAddr, error) {
	ip := net.ParseIP(c.ClientAddr)
	if ip == nil {
		return nil, fmt.Errorf("Failed to parse IP: %v", c.ClientAddr)
	}
	return &net.TCPAddr{IP: ip, Port: port}, nil
}

// DecodeConfig reads the configuration from the given reader in JSON
// format and decodes it into a proper Config structure.
func DecodeConfig(r io.Reader) (*Config, error) {
	var raw interface{}
	var result Config
	dec := json.NewDecoder(r)
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}

	// Check the result type
	if obj, ok := raw.(map[string]interface{}); ok {
		// Check for a "service" or "check" key, meaning
		// this is actually a definition entry
		/*if sub, ok := obj["service"]; ok {
			service, err := DecodeServiceDefinition(sub)
			result.Services = append(result.Services, service)
			return &result, err
		} else if sub, ok := obj["check"]; ok {
			check, err := DecodeCheckDefinition(sub)
			result.Checks = append(result.Checks, check)
			return &result, err
		}*/
	}

	// Decode
	var md mapstructure.Metadata
	msdec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: &md,
		Result:   &result,
	})
	if err != nil {
		return nil, err
	}

	if err := msdec.Decode(raw); err != nil {
		return nil, err
	}

	return &result, nil
}


// MergeConfig merges two configurations together to make a single new
// configuration.
func MergeConfig(a, b *Config) *Config {
	var result Config = *a

	// Copy the strings if they're set
	if b.LogLevel != "" {
		result.LogLevel = b.LogLevel
	}
	if b.Protocol > 0 {
		result.Protocol = b.Protocol
	}
	if b.ClientAddr != "" {
		result.ClientAddr = b.ClientAddr
	}
	if b.BindAddr != "" {
		result.BindAddr = b.BindAddr
	}
	if b.AdvertiseAddr != "" {
		result.AdvertiseAddr = b.AdvertiseAddr
	}
	if b.Server == true {
		result.Server = b.Server
	}
	if b.LeaveOnTerm == true {
		result.LeaveOnTerm = true
	}
	if b.SkipLeaveOnInt == true {
		result.SkipLeaveOnInt = true
	}
	if b.EnableDebug {
		result.EnableDebug = true
	}
	if b.VerifyIncoming {
		result.VerifyIncoming = true
	}
	if b.VerifyOutgoing {
		result.VerifyOutgoing = true
	}
	if b.CAFile != "" {
		result.CAFile = b.CAFile
	}
	if b.CertFile != "" {
		result.CertFile = b.CertFile
	}
	if b.KeyFile != "" {
		result.KeyFile = b.KeyFile
	}
	if b.Ports.HTTP != 0 {
		result.Ports.HTTP = b.Ports.HTTP
	}
	if b.Ports.RPC != 0 {
		result.Ports.RPC = b.Ports.RPC
	}
	if b.Ports.Server != 0 {
		result.Ports.Server = b.Ports.Server
	}
	if b.UiDir != "" {
		result.UiDir = b.UiDir
	}
	if b.PidFile != "" {
		result.PidFile = b.PidFile
	}

	return &result
}

// ReadConfigPaths reads the paths in the given order to load configurations.
// The paths can be to files or directories. If the path is a directory,
// we read one directory deep and read any files ending in ".json" as
// configuration files.
func ReadConfigPaths(paths []string) (*Config, error) {
	result := new(Config)
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("Error reading '%s': %s", path, err)
		}

		fi, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("Error reading '%s': %s", path, err)
		}

		if !fi.IsDir() {
			config, err := DecodeConfig(f)
			f.Close()

			if err != nil {
				return nil, fmt.Errorf("Error decoding '%s': %s", path, err)
			}

			result = MergeConfig(result, config)
			continue
		}

		contents, err := f.Readdir(-1)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("Error reading '%s': %s", path, err)
		}

		// Sort the contents, ensures lexical order
		sort.Sort(dirEnts(contents))

		for _, fi := range contents {
			// Don't recursively read contents
			if fi.IsDir() {
				continue
			}

			// If it isn't a JSON file, ignore it
			if !strings.HasSuffix(fi.Name(), ".json") {
				continue
			}

			subpath := filepath.Join(path, fi.Name())
			f, err := os.Open(subpath)
			if err != nil {
				return nil, fmt.Errorf("Error reading '%s': %s", subpath, err)
			}

			config, err := DecodeConfig(f)
			f.Close()

			if err != nil {
				return nil, fmt.Errorf("Error decoding '%s': %s", subpath, err)
			}

			result = MergeConfig(result, config)
		}
	}

	return result, nil
}

// Implement the sort interface for dirEnts
func (d dirEnts) Len() int {
	return len(d)
}

func (d dirEnts) Less(i, j int) bool {
	return d[i].Name() < d[j].Name()
}

func (d dirEnts) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
