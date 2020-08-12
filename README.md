# sftr - Simple File Transfer Regent

`sftr` acts as a basic gatekeeper between a client that wants to upload or
download a file and a login account on a server running an appropriate SSH
daemon.  Instead of needing to run another service such as FTP or allowing
arbitrary uploads/downloads, `sftr` is invoked on behalf of a user and limits
transfers to a specific list and specific clients, and can automatically run
commands after transfers (coming soon).

## Use with OpenSSH

`sftr` is intended for use as a [forced
command](https://man.openbsd.org/sshd#command=_command_) associated with an
SSH key.  As the name implies, associating a program with an SSH key in this
way causes the program to be invoked on use of that key regardless of the
actual command supplied.

Generate a new, dedicated SSH key and place the public component in
`~/.ssh/authorized_keys` as you normally would.  Prepend the line with
recommended options including the `sftr` command:

```
command="/opt/computecanada/bin/sftr --config=$HOME/.sftr",no-port-forwarding,no-X11-forwarding,no-pty ssh-ed25519
<key> deploy-bot
```

Note this should all be a single line and the first field cannot contain any
spaces except as enclosed with quotes.

## Configuration

Configuration for `sftr` is YAML and lists targets for deposit or retrieval,
and suitable clients:

```
---
resources:
  - paths: ['/etc/hosts']
    op: get
    from: 192.168.14.0/24
  - paths: ['/var/repo/backup']
    op: put
    from: 192.168.16.233/32
```

In this example, `sftr` will allow a host in the 192.168.14.0/24 network to
retrieve the hosts file, and a specific host to upload its backup.  Note the
addresses must use full CIDR notation, and so must include `/32` for
individual hosts.

## Trying it out

There are some things to watch out for when trying this out.

### Disable your SSH authentication agent

If you're using an SSH authentication agent, the registered identities may
conflict with the specific key you want to use, and you'll get odd results
(specifically, the shell failling to execute `put testfile`).  Here's a way
around that:

```
$ SSH_AUTH_SOCK=/dev/null ssh -i ~/.ssh/test-deploy myhost put testfile
```

## Future

This is an initial release.  Planned future enhancements:

* proper logging, robustness, better error checking
* post-transfer script support
* client and client group definitions for more readable and less error-prone
  access configuration
* support for patterns and/or globs in filenames, and directories

## Alternatives

A specific SSH key could be tied to other programs you already have, or simple
shell scripts.  These may fit your needs, but here are where they can fall
short:

* One-shot script that does the same thing (accepts input on stdin, writes to
  target file, maybe does some post-processing afterwards.  Or the reverse):
  this is certainly doable, and this program was written so I wouldn't have to
  keep doing that.

* rsync - oddly, not all systems actually have this.  It can be used in a lot
  of cases.  I recommend using a basic script to log arguments received as the
  forced command, in order to determine what options rsync will be expecting on
  the server side.  These will need to be reproduced carefully, if
  incompletely, but enough so that the client and server can talk.  If done
  right, rsync can fulfill a lot of needs.

* scp/sftp.  Could be used to transfer files and SSH used to follow up to run
  specific post commands.  This will be more efficient in transferring multiple
  files, but you can't limit what files are transferred, except the normal
  permissions structures available on the remote system.

* ftp.  Still a thing?  Ugh.
