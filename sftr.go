package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
)

// --------------------------------------------------------------------------
//                                                                constants
// --------------------------------------------------------------------------

// error codes
const ERR_NO_MATCH = 1
const ERR_INVOCATION = 128
const ERR_CONFIGURATION = 129
const ERR_FILE_OPEN = 130
const ERR_FILE_IO = 131
const ERR_POSTEXEC = 132

// program details
const PROGRAM_NAME = "sftr"
const PROGRAM_VERSION = "dev"
const PROGRAM_USAGE = `
Usage: sftr -help         for this help text, or
       sftr [options]

It is assumed this program is used as a forced command as described in sshd(8)
and $SSH_ORIGINAL_COMMAND and $SSH_CONNECTION are set in the environment.
$SSH_ORIGINAL_COMMAND must consist of an operation and an operand.  Depending
on whether the operations is "put" or "get", writes to or reads from the file
indicated by the operand, using standard input and output as appropriate.

Options:
`

// --------------------------------------------------------------------------
//                                                                structures
// --------------------------------------------------------------------------

type Client struct {
	// undefined for now
}

type Resource struct {
	Paths   []string
	Op      string
	From    string
	Command []string
	Script  string
}

type Config struct {
	Clients   []Client
	Resources []Resource
}

type Args struct {
	config_fn string
	syslog    bool
}

// --------------------------------------------------------------------------
//                                                                functions
// --------------------------------------------------------------------------

// Parse arguments and handle usage display.  Return struct describing the
// arguments.
func parse_cli() Args {
	var args Args

	config_fn := PROGRAM_NAME + ".yaml"
	flag.StringVar(&args.config_fn, "config", config_fn, "config file")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), PROGRAM_USAGE)
		flag.PrintDefaults()
	}

	flag.Parse()
	return args
}

// Read config from given filename, so long as it adheres to expected YAML
// structure.  Return struct representing that config.
func get_config(filename string) (config Config) {

	// read configuration file
	data, err := ioutil.ReadFile(filename)
	check(ERR_INVOCATION, err)

	// unmarshal into struct
	err = yaml.UnmarshalStrict([]byte(data), &config)
	check(ERR_CONFIGURATION, err)

	return
}

// Parse client information from environment variables provided by SSH daemon:
// retrieve client IP from connection details and interpret command supplied
// on the client side as an operation and target path.  Return client IP,
// operation and path.
func get_ssh_info() (from net.IP, op, path string) {

	// get SSH connection information
	ssh_connection, isset := os.LookupEnv("SSH_CONNECTION")
	if !isset {
		fatal(ERR_INVOCATION, "No SSH connection information available")
	}

	// TODO: make more robust
	from = net.ParseIP(strings.Split(ssh_connection, " ")[0])
	if from == nil {
		fatal(ERR_INVOCATION, "Could not parse originating IP address from SSH connection information")
	}

	// operation is from command passed to SSH call
	ssh_command, isset := os.LookupEnv("SSH_ORIGINAL_COMMAND")
	if !isset {
		fatal(ERR_INVOCATION, "No operation supplied")
	}

	// split operation into op + erand
	ops := strings.Split(ssh_command, " ")[0:2]
	op = ops[0]
	path = ops[1]
	if op == "" || path == "" {
		fatal(ERR_INVOCATION, "Must specify op and path")
	} else if op != "get" && op != "put" {
		fatal(ERR_INVOCATION, "Op '%s' not valid", op)
	}

	return
}

// Search through configuration for first resource definition matching
// incoming client IP, desired operation and target.  Return index of found
// entry, matching resource, and a boolean indicating whether those two values
// have any meaning.
func find_first(ip net.IP, op, path string, config Config) (idx int, res Resource, found bool) {

	found = false
	for idx, res = range config.Resources {
		if op == res.Op &&
			(string_occurs_in_array(path, res.Paths) ||
				string_matches_glob_in_array(path, res.Paths)) {

			// parse "from" into network
			_, ipnet, err := net.ParseCIDR(res.From)
			check(ERR_CONFIGURATION, err)

			// test whether client IP matches this network
			if ipnet.Contains(ip) {
				found = true
				return
			}
		}
	}

	return
}

// --------------------------------------------------------------------------
//                                                                main
// --------------------------------------------------------------------------

func main() {

	// get flags - first thing in case of invocation error or usage
	args := parse_cli()

	info("Starting " + PROGRAM_NAME)

	// get runtime parameters from SSH environment variables
	from, op, path := get_ssh_info()

	// parse configuration file
	config := get_config(args.config_fn)

	// verify what was specified through SSH matches an entry
	_, resource, found := find_first(from, op, path, config)
	if !found {
		fatal(ERR_NO_MATCH, "No such match for %s", from)
	}

	// now do the thing
	if op == "put" {
		// open target
		fp, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0640)
		check(ERR_FILE_OPEN, err)
		defer func() {
			fp.Close()
			info("Closed target file")
		}()

		// read from stdin and write to target
		_, err = io.Copy(fp, os.Stdin)
		check(ERR_FILE_IO, err)

		// if post-command specified
		if len(resource.Command) != 0 {
			// TODO: split this out
			info("Found command: %s", resource.Command)

			// set up command structure (split program and arguments)
			cmd := exec.Command(resource.Command[0], resource.Command[1:]...)

			// run command and get output
			output, err := cmd.CombinedOutput()
			check(ERR_POSTEXEC, err)
			info("Command: %s", output)
		} else if resource.Script != "" {
			// TODO: split this out
			// TODO: make shell configurable
			cmd := exec.Command("/bin/sh")

			// create input pipe for command
			stdin, err := cmd.StdinPipe()
			check(ERR_POSTEXEC, err)

			// write to pipe
			// from (the example)[https://golang.org/pkg/os/exec/#Cmd.StdinPipe]
			go func() {
				defer stdin.Close()
				io.WriteString(stdin, resource.Script)
			}()

			// get the output
			output, err := cmd.CombinedOutput()
			check(ERR_POSTEXEC, err)
			info("Script: %s", output)
		}
	} else if op == "get" {
		// open source
		fp, err := os.Open(path)
		check(ERR_FILE_OPEN, err)
		defer func() {
			fp.Close()
			info("Closed source file")
		}()

		// read from source and write to stdout
		_, err = io.Copy(os.Stdout, fp)
		check(ERR_FILE_IO, err)
	} else {
		fatal(ERR_INVOCATION, "Unsupported operation '%s'", op)
	}
}
