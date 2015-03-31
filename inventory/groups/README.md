# Groups
Type definitions and functions to work with ansible groups within a rethink db.

Please refer to http://rethinkdb.com/docs/security/ for secure config of rethink db

Especially use the following to test remote connections via ssh tunnel
ssh -L {{ local_port }}:localhost:{{ remote_port }} {{ remote_ssh_user }}@{{ remote_server }} -p{{ remote_ssh_port }}