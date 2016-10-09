Reservation
----

Mesos resources can be reserved to a role.
There are two kinds of reservation: _Static Reservation_ and _Dynamic Reservation_.

# Static Reservation

Mesos agent can be configured to reserve some resources for a role.

## Demo

This Demo will reserve 0.5 cpus and 200M mem for role `roleA` on agent 192.168.56.21.

Start agent on 192.168.56.21 with these configurations(`etc/mesos-agent-env.sh`):
```
export MESOS_RESOURCES="cpus:1;mem:1000;cpus(roleA):0.5;mem(roleA):200"
```

Start simpleScheduler as `roleA` and print offers:
```
$ ./simple_scheduler -host=192.168.56.11 -master=192.168.56.23:5050 -role roleA -justPrintOffers
```

we can find resources from slave `192.168.56.21` like this:
```
...
    "resources": [
      {"name": "cpus", "type": 0, "scalar": {"value": 0.5}, "role": "roleA"},
      {"name": "mem", "type": 0, "scalar": {"value": 200}, "role": "roleA"},
      {"name": "cpus","type": 0, "scalar": {"value": 1}, "role": "*"},
      {"name": "mem", "type": 0, "scalar": {"value": 1000}, "role": "*"},
      {"name": "disk", "type": 0, "scalar": {"value": 58503}, "role": "*"},
      {"name": "ports", "type": 1, "ranges": {"range": [{"begin": 31000, "end": 32000}]}, "role": "*"}
    ]
...
```

We can find two parts of resources , one for all roles(*), one for only `roleA`.

Start simpleScheduler as `roleB` and print offers:
```
$ ./simple_scheduler -host=192.168.56.11 -master=192.168.56.23:5050 -role roleB -justPrintOffers
```

Offered resources from 192.168.56.21:
```
    "resources": [
      {"name": "cpus", "type": 0, "scalar": {"value": 1 }, "role": "*" },
      {"name": "mem", "type": 0, "scalar": {"value": 1000 }, "role": "*" },
      {"name": "disk", "type": 0, "scalar": {"value": 58503 }, "role": "*" },
      {"name": "ports", "type": 1, "ranges": {"range": [{"begin": 31000, "end": 32000}]}, "role": "*"}
    ],
```

We can only get resources for all roles.


# Dynamic Reservation

## Demo

SimpleScheduler will dynamically reserve 0.5 cpus and 200 mem on some host for role `roleC`.

I try to reserve some resource from framework, but got an error:
```
E0906 14:34:41.592005    5049 messenger.go:318] failed to send message "scheduler": master master@192.168.56.21:5050 rejected /api/v1/scheduler, returned status "403 Forbidden"
```
Looks like somethings went wrong in the `mesos-go` pkg.
