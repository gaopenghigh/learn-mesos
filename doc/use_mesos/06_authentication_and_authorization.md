Authentication and Authorization
----

# Authentication

Authentication permits only trusted entities to interact with a Mesos cluster.
Authentication is disabled by default.

## Credentials, Principles and Secrets

Entities that wants to authenticate with Mesos must provide a _credential_, which consists of a _principal_ and a
_secret_. Principals are similar to user names, while secrets are similar to passwords.

# Authorization

Authorization subsystem of Mesos allows the operator to configure the actions that certain principals are allowed to
perform.

## Roles and Weights

**Roles** can be used to specify that certain resources are reserved for the use of one or more frameworks.
Roles are configured using Access Control Lists(ACLs).

**Weights** can be used to control the relative share of cluster resources that is offered to different roles.

## Role vs. Principal

 A useful analogy can be made with user management in the Unix world: principals correspond to users, while roles
 approximately correspond to groups.
 
# Configuration

See [Mesos Document](http://mesos.apache.org/documentation/latest/authorization/) for more ACL configuration examples.

# Demo

Here is a simple roles and weights demo.

Suppose we have tow roles:
* `roleA`, with weight 1.0
* `roleB`, with weight 10.0 

Create configuration file `weights.json`:
```
    [
      {
        "role": "roleA",
        "weight": 1.0
      },
      {
        "role": "roleB",
        "weight": 2.0
      }
    ]
```

Start Mesos cluster, update weights configuration:
```
$ curl -X PUT -d @weights.json http://192.168.56.21:5050/weights

$ curl -X GET http://192.168.56.21:5050/weights
[{"role":"roleB","weight":2.0},{"role":"roleA","weight":1.0}]
```

Start 2 schedulers with `roleA` and `roleB`:

Codes:
```
func newSimpleScheduler() *simpleScheduler {
	s := &simpleScheduler{
		shutdown: make(chan struct{}),
		shellCmdQueue: list.New(),
	}
	// s.shellCmdQueue.PushBack("cat /etc/issue && ps aux && env && while true; do echo command running; sleep 3; done")
	for i := 0; i < 10000; i ++ {
		 s.shellCmdQueue.PushBack("ps aux && env && sleep 10 && echo finished")
	}

	return s
}

...

func (s *simpleScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	s.runCommandTasks(driver, offers, s.newShellCommandTask)
}
```

Commands:
```
# terminal 1
$ ./simple_scheduler -master "192.168.56.21:5050" -host "192.168.56.11" -role "roleA"

# terminal 2
$ ./simple_scheduler -master "192.168.56.21:5050" -host "192.168.56.11" -role "roleB"
```

Wait two schedulers to run severial minutes.
Check resource usage by visit Mesos master API: `http://192.168.56.21:5050/frameworks`:
* Framework with `roleA(weight=1.0)`, has 12 running tasks, 90 completed tasks, total 102
* Framework with `roleB(weight=2.0)`, has 18 running tasks, 202 completed tasks, total 220

Resources offered to `roleA` is almost 1/2 of resources offered to `roleB`, this is what we expected.
