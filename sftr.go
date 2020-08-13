package main

import (
	"flag"
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

const ERR_NO_MATCH = 1
const ERR_INVOCATION = 128
const ERR_CONFIGURATION = 129
const ERR_FILE_OPEN = 130
const ERR_FILE_IO = 131
const ERR_POSTEXEC = 132

// --------------------------------------------------------------------------
//                                                                structures
// --------------------------------------------------------------------------

type ClientDef struct {
	// undefined for now
}

type ResourceDef struct {
	Paths   []string
	Op      string
	From    string
	Command []string
	Script  string
}

type Config struct {
	Clients   []ClientDef
	Resources []ResourceDef
}

// --------------------------------------------------------------------------
//                                                                main
// --------------------------------------------------------------------------

func main() {
	var config Config

	info("Starting sftr")

	// get SSH connection information
	var ssh_connection string
	var isset bool
	ssh_connection, isset = os.LookupEnv("SSH_CONNECTION")
	if !isset {
		fatal(ERR_INVOCATION, "No SSH connection information available")
	}

	// make more robust
	from_ip := net.ParseIP(strings.Split(ssh_connection, " ")[0])
	if from_ip == nil {
		fatal(ERR_INVOCATION, "Could not parse originating IP address from SSH connection information")
	}
	info("Coming from %s", from_ip)

	// get flags
	configPtr := flag.String("config", "sftr.yaml", "config file")
	flag.Parse()

	// operation is from command passed to SSH call
	var ssh_command string
	ssh_command, isset = os.LookupEnv("SSH_ORIGINAL_COMMAND")
	if !isset {
		fatal(ERR_INVOCATION, "No operation supplied")
	}

	// split operation into op + erand
	ops := strings.Split(ssh_command, " ")[0:2]
	op := ops[0]
	path := ops[1]
	if op == "" || path == "" {
		fatal(ERR_INVOCATION, "Must specify op and path")
	}
	info("op = %s, path = %s", op, path)

	// read configuration file
	// TODO: configurable also via environment variable
	data, err := ioutil.ReadFile(*configPtr)
	check(ERR_INVOCATION, err)

	// unmarshal into struct
	err = yaml.UnmarshalStrict([]byte(data), &config)
	check(ERR_CONFIGURATION, err)

	// verify op and path match an entry
	var idx int
	var found = false
	var resource ResourceDef
	for idx, resource = range config.Resources {
		if op == resource.Op &&
			(string_occurs_in_array(path, resource.Paths) ||
				string_matches_glob_in_array(path, resource.Paths)) {

			// parse "from" into network
			_, ipnet, err := net.ParseCIDR(resource.From)
			check(ERR_CONFIGURATION, err)

			// test whether client IP matches this network
			debug("Checking whether %s contains %s", ipnet, from_ip)
			if ipnet.Contains(from_ip) {
				found = true
				break
			}
		}
	}

	if !found {
		fatal(ERR_NO_MATCH, "No such match for %s", from_ip)
	}

	info("Found matching resource at index %d", idx)

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
