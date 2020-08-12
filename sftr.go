package main

import (
  "flag"
  "log"
  "io"
  "io/ioutil"
  "os"
  "gopkg.in/yaml.v2"
  "strings"
  "net"
)

type ClientDef struct {
  // undefined for now
}

type ResourceDef struct {
  Paths []string
  Op string
  From string
}

type Config struct {
  Clients []ClientDef
  Resources []ResourceDef
}

func check(e error) {
  if e != nil {
    log.Fatal(e)
    os.Exit(1)
  }
}

func sina(s string, a []string) bool {
  for _, c := range a {
    if c == s {
      return true
    }
  }
  return false
}

func main() {
  var config Config

  log.Println("Starting sftr")

  // get SSH connection information
  var ssh_connection string
  var isset bool
  ssh_connection, isset = os.LookupEnv("SSH_CONNECTION")
  if !isset {
    log.Fatal("No SSH connection information available")
    os.Exit(1)
  }

  // make more robust
  from_ip := net.ParseIP(strings.Split(ssh_connection, " ")[0])
  if from_ip == nil {
    log.Fatal("Could not parse originating IP address from SSH connection information")
    os.Exit(1)
  }
  log.Printf("Coming from %s", from_ip)

  // get flags
  configPtr := flag.String("config", "sftr.yaml", "config file")
  flag.Parse()

  // operation is from command passed to SSH call
  var ssh_command string
  ssh_command, isset = os.LookupEnv("SSH_ORIGINAL_COMMAND")
  if !isset {
    log.Fatal("No operation supplied")
    os.Exit(1)
  }

  // split operation into op + erand
  ops := strings.Split(ssh_command, " ")[0:2]
  op := ops[0]
  path := ops[1]
  if op == "" || path == "" {
    log.Fatal("Must specify op and path")
    os.Exit(1)
  }
  log.Printf("op = %s, path = %s", op, path)

  // read configuration file
  // TODO: configurable also via environment variable
  data, err := ioutil.ReadFile(*configPtr)
  check(err)

  // unmarshal into struct
  err = yaml.UnmarshalStrict([]byte(data), &config)
  check(err)

  // verify op and path match an entry
  var idx int
  var found = false
  var resource ResourceDef
  for idx, resource = range config.Resources {
    if op == resource.Op && sina(path, resource.Paths) {

      // parse "from" into network
      _, ipnet, err := net.ParseCIDR(resource.From)
      check(err)

      // test whether client IP matches this network
      log.Printf("Checking whether %s contains %s", ipnet, from_ip)
      if ipnet.Contains(from_ip) {
        found = true
      }
    }
  }

  if !found {
    log.Fatal("None such, sorry")
    os.Exit(1)
  }

  log.Printf("Found matching resource at index %d", idx)

  // now do the thing
  if op == "put" {
    // open target
    fp, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0640) 
    check(err)

    // read from stdin and write to target
    _, err = io.Copy(fp, os.Stdin)
    check(err)
  } else if op == "get" {
    // open source
    fp, err := os.Open(path)
    check(err)

    // read from source and write to stdout
    _, err = io.Copy(os.Stdout, fp)
    check(err)
  } else {
    log.Fatal("Unsupported operation '%s'", op)
    os.Exit(1)
  }
}
