package agent

import (
	"flag"
	"fmt"
	"github.com/armon/go-metrics"
	"github.com/hashicorp/logutils"
	"github.com/mitchellh/cli"
	"io"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// gracefulTimeout controls how long we wait before forcefully terminating
var gracefulTimeout = 5 * time.Second

// Command is a Command implementation that runs a Consul agent.
// The command will not end unless a shutdown message is sent on the
// ShutdownCh. If two messages are sent on the ShutdownCh it will forcibly
// exit.
type Command struct {
	Ui         cli.Ui
	ShutdownCh <-chan struct{}
	args       []string
	logFilter  *logutils.LevelFilter
	agent      *Agent
	httpServer *HTTPServer
}

// readConfig is responsible for setup of our configuration using
// the command line and any file configs
func (c *Command) readConfig() *Config {
	var cmdConfig Config
	var configFiles []string
	cmdFlags := flag.NewFlagSet("agent", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	//cmdFlags.Var((*AppendSliceValue)(&configFiles), "config-file", "json file to read config from")
	//cmdFlags.Var((*AppendSliceValue)(&configFiles), "config-dir", "directory of json files to read")

	cmdFlags.StringVar(&cmdConfig.LogLevel, "log-level", "", "log level")
	//cmdFlags.StringVar(&cmdConfig.DataDir, "data-dir", "", "path to the data directory")
	cmdFlags.StringVar(&cmdConfig.UiDir, "ui-dir", "", "path to the web UI directory")
	cmdFlags.StringVar(&cmdConfig.PidFile, "pid-file", "", "path to file to store PID")

	cmdFlags.BoolVar(&cmdConfig.Server, "server", false, "run agent as server")

	cmdFlags.StringVar(&cmdConfig.ClientAddr, "client", "", "address to bind client listeners to (DNS, HTTP, RPC)")
	cmdFlags.StringVar(&cmdConfig.BindAddr, "bind", "", "address to bind server listeners to")

	cmdFlags.IntVar(&cmdConfig.Protocol, "protocol", -1, "protocol version")

	if err := cmdFlags.Parse(c.args); err != nil {
		return nil
	}

	config := DefaultConfig()
	if len(configFiles) > 0 {
		fileConfig, err := ReadConfigPaths(configFiles)
		if err != nil {
			c.Ui.Error(err.Error())
			return nil
		}

		config = MergeConfig(config, fileConfig)
	}

	config = MergeConfig(config, &cmdConfig)

	// Warn if using windows as a server
	if config.Server && runtime.GOOS == "windows" {
		c.Ui.Error("WARNING: Windows is not recommended as a Consul server. Do not use in production.")
	}

	return config
}

// setupLoggers is used to setup the logGate, logWriter, and our logOutput
func (c *Command) setupLoggers(config *Config) (*GatedWriter, *logWriter, io.Writer) {
	// Setup logging. First create the gated log writer, which will
	// store logs until we're ready to show them. Then create the level
	// filter, filtering logs of the specified level.
	logGate := &GatedWriter{
		Writer: &cli.UiWriter{Ui: c.Ui},
	}

	c.logFilter = LevelFilter()
	c.logFilter.MinLevel = logutils.LogLevel(strings.ToUpper(config.LogLevel))
	c.logFilter.Writer = logGate
	if !ValidateLevelFilter(c.logFilter.MinLevel, c.logFilter) {
		c.Ui.Error(fmt.Sprintf(
			"Invalid log level: %s. Valid log levels are: %v",
			c.logFilter.MinLevel, c.logFilter.Levels))
		return nil, nil, nil
	}

	// Create a log writer, and wrap a logOutput around it
	logWriter := NewLogWriter(512)
	logOutput := io.MultiWriter(c.logFilter, logWriter)
	return logGate, logWriter, logOutput
}

// setupAgent is used to start the agent and various interfaces
func (c *Command) setupAgent(config *Config, logOutput io.Writer, logWriter *logWriter) error {
	c.Ui.Output("Starting Consul agent...")
	agent, err := Create(config, logOutput)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error starting agent: %s", err))
		return err
	}
	c.agent = agent

	// Setup the RPC listener
	rpcAddr, err := config.ClientListener(config.Ports.RPC)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Invalid RPC bind address: %s", err))
		return err
	}

	rpcListener, err := net.Listen("tcp", rpcAddr.String())
	if err != nil {
		agent.Shutdown()
		c.Ui.Error(fmt.Sprintf("Error starting RPC listener: %s", err))
		return err
	}

	// Start the IPC layer
	//c.Ui.Output("Starting Consul agent RPC...")
	//c.rpcServer = NewAgentRPC(agent, rpcListener, logOutput, logWriter)

	if config.Ports.HTTP > 0 {
		httpAddr, err := config.ClientListener(config.Ports.HTTP)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Invalid HTTP bind address: %s", err))
			return err
		}

		server, err := NewHTTPServer(agent, config.UiDir, config.EnableDebug, logOutput, httpAddr.String())
		if err != nil {
			agent.Shutdown()
			c.Ui.Error(fmt.Sprintf("Error starting http server: %s", err))
			return err
		}
		c.httpServer = server
	}

	return nil
}


func (c *Command) Run(args []string) int {
	c.Ui = &cli.PrefixedUi{
		OutputPrefix: "==> ",
		InfoPrefix:   "    ",
		ErrorPrefix:  "==> ",
		Ui:           c.Ui,
	}

	// Parse our configs
	c.args = args
	config := c.readConfig()
	if config == nil {
		return 1
	}
	c.args = args

	// Check GOMAXPROCS
	if runtime.GOMAXPROCS(0) == 1 {
		c.Ui.Error("WARNING: It is highly recommended to set GOMAXPROCS higher than 1")
	}

	// Setup the log outputs
	logGate, logWriter, logOutput := c.setupLoggers(config)
	if logWriter == nil {
		return 1
	}

	/* Setup telemetry
	Aggregate on 10 second intervals for 1 minute. Expose the
	metrics over stderr when there is a SIGUSR1 received.
	*/
	inm := metrics.NewInmemSink(10*time.Second, time.Minute)
	metrics.DefaultInmemSignal(inm)
	metricsConf := metrics.DefaultConfig("consul")

	// Optionally configure a statsite sink if provided
	if config.StatsiteAddr != "" {
		sink, err := metrics.NewStatsiteSink(config.StatsiteAddr)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to start statsite sink. Got: %s", err))
			return 1
		}
		fanout := metrics.FanoutSink{inm, sink}
		metrics.NewGlobal(metricsConf, fanout)

	} else {
		metricsConf.EnableHostname = false
		metrics.NewGlobal(metricsConf, inm)
	}

	// Create the agent
	if err := c.setupAgent(config, logOutput, logWriter); err != nil {
		return 1
	}
	defer c.agent.Shutdown()
	//if c.rpcServer != nil {
	//	defer c.rpcServer.Shutdown()
	//}
	if c.httpServer != nil {
		defer c.httpServer.Shutdown()
	}


	c.Ui.Output("Enforcer server running!")
	c.Ui.Info(fmt.Sprintf(" Client Addr: %v (HTTP: %d, DNS: %d, RPC: %d)", config.ClientAddr,
		config.Ports.HTTP, config.Ports.RPC))

	// Enable log streaming
	c.Ui.Info("")
	c.Ui.Output("Log data will now stream in as it occurs:\n")
	logGate.Flush()

	// Wait for exit
	return c.handleSignals(config)
}

// handleSignals blocks until we get an exit-causing signal
func (c *Command) handleSignals(config *Config) int {
	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	// Wait for a signal
WAIT:
	var sig os.Signal
	select {
	case s := <-signalCh:
		sig = s
	case <-c.ShutdownCh:
		sig = os.Interrupt
	case <-c.agent.ShutdownCh():
		// Agent is already shutdown!
		return 0
	}
	c.Ui.Output(fmt.Sprintf("Caught signal: %v", sig))

	// Check if this is a SIGHUP
	if sig == syscall.SIGHUP {
		config = c.handleReload(config)
		goto WAIT
	}

	// Check if we should do a graceful leave
	graceful := false
	if sig == os.Interrupt && !config.SkipLeaveOnInt {
		graceful = true
	} else if sig == syscall.SIGTERM && config.LeaveOnTerm {
		graceful = true
	}

	// Bail fast if not doing a graceful leave
	if !graceful {
		return 1
	}

	// Attempt a graceful leave
	gracefulCh := make(chan struct{})
	c.Ui.Output("Gracefully shutting down agent...")
	go func() {
		if err := c.agent.Leave(); err != nil {
			c.Ui.Error(fmt.Sprintf("Error: %s", err))
			return
		}
		close(gracefulCh)
	}()

	// Wait for leave or another signal
	select {
	case <-signalCh:
		return 1
	case <-time.After(gracefulTimeout):
		return 1
	case <-gracefulCh:
		return 0
	}
}

// handleReload is invoked when we should reload our configs, e.g. SIGHUP
func (c *Command) handleReload(config *Config) *Config {
	c.Ui.Output("Reloading configuration...")
	newConf := c.readConfig()
	if newConf == nil {
		c.Ui.Error(fmt.Sprintf("Failed to reload configs"))
		return config
	}

	// Change the log level
	minLevel := logutils.LogLevel(strings.ToUpper(newConf.LogLevel))
	if ValidateLevelFilter(minLevel, c.logFilter) {
		c.logFilter.SetMinLevel(minLevel)
	} else {
		c.Ui.Error(fmt.Sprintf(
			"Invalid log level: %s. Valid log levels are: %v",
			minLevel, c.logFilter.Levels))

		// Keep the current log level
		newConf.LogLevel = config.LogLevel
	}

	return newConf
}


func (c *Command) Synopsis() string {
	return "Runs a Consul agent"
}

func (c *Command) Help() string {
	helpText := `
Usage: consul agent [options]

  Starts the Consul agent and runs until an interrupt is received. The
  agent represents a single node in a cluster.

Options:

  -bootstrap               Sets server to bootstrap mode
  -bind=0.0.0.0            Sets the bind address for cluster communication
  -client=127.0.0.1        Sets the address to bind for client access.
                           This includes RPC, DNS and HTTP
  -config-file=foo         Path to a JSON file to read configuration from.
                           This can be specified multiple times.
  -config-dir=foo          Path to a directory to read configuration files
                           from. This will read every file ending in ".json"
                           as configuration in this directory in alphabetical
                           order.
  -data-dir=path           Path to a data directory to store agent state
  -dc=east-aws             Datacenter of the agent
  -join=1.2.3.4            Address of an agent to join at start time.
                           Can be specified multiple times.
  -log-level=info          Log level of the agent.
  -node=hostname           Name of this node. Must be unique in the cluster
  -protocol=N              Sets the protocol version. Defaults to latest.
  -server                  Switches agent to server mode.
  -ui-dir=path             Path to directory containing the Web UI resources

 `
	return strings.TrimSpace(helpText)
}
