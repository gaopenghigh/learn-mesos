Basic Scheduler
----

I will write a very simple scheduler to show what offers look like.

# Install Mesos

I installed a 3-nodes Mesos cluster, 3 Mesos masters and 3 Mesos agents, IPs:
```
192.168.56.21, 192.168.56.22, 192.168.56.23
```

# Framework Register

```
func (s *simpleScheduler) Registered(
	_ sched.SchedulerDriver,
	frameworkID *mesos.FrameworkID,
	masterInfo *mesos.MasterInfo) {
	log.WithFields(log.Fields{"frameworkID": frameworkID, "masterInfo": masterInfo}).Info("framework registered")
}
```

Logs:

```
{
  "frameworkID": {
    "value": "8e9bd182-6693-493a-8edb-4c7e2cf90bbd-0013"
  },
  "level": "info",
  "masterInfo": {
    "id": "8e9bd182-6693-493a-8edb-4c7e2cf90bbd",
    "ip": 356034752,
    "port": 5050,
    "pid": "master@192.168.56.21:5050",
    "hostname": "192.168.56.21",
    "version": "1.0.0",
    "address": {
      "hostname": "192.168.56.21",
      "ip": "192.168.56.21",
      "port": 5050
    }
  },
  "msg": "framework registered",
  "time": "2016-08-11T16:25:01+08:00"
}
```

# Receive Offers:

Do nothing but print out offer the framework got.

```
func (s *simpleScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	s.printOffers(offers)
}

func (s *simpleScheduler) printOffers(offers []*mesos.Offer) {
	log.Printf("Received %d resource offers", len(offers))
	for _, offer := range offers {
		log.WithFields(log.Fields{"offer": offer.String()}).Info("offer")
	}
}
```

Logs:

```
{
  "level":"info",
  "msg": "Received 3 resource offers",
  "time":"2016-08-11T16:29:03+08:00"
}

{
  "level": "info",
  "msg": "offer",
  "offer": {
    "id": {
      "value": "8e9bd182-6693-493a-8edb-4c7e2cf90bbd-O30"
    },
    "framework_id": {
      "value": "8e9bd182-6693-493a-8edb-4c7e2cf90bbd-0014"
    },
    "slave_id": {
      "value": "f6f7b9a4-235d-4855-877f-a49e62294c2b-S2"
    },
    "hostname": "192.168.56.22",
    "url": {
      "scheme": "http",
      "address": {
        "hostname": "192.168.56.22",
        "ip": "192.168.56.22",
        "port": 5051
      },
      "path": "/slave(1)"
    },
    "resources": [
      {
        "name": "cpus",
        "type": 0,
        "scalar": {
          "value": 0.5
        },
        "role": "*"
      },
      {
        "name": "mem",
        "type": 0,
        "scalar": {
          "value": 464
        },
        "role": "*"
      },
      {
        "name": "disk",
        "type": 0,
        "scalar": {
          "value": 58503
        },
        "role": "*"
      },
      {
        "name": "ports",
        "type": 1,
        "ranges": {
          "range": [
            {
              "begin": 31000,
              "end": 31054
            },
            {
              "begin": 31056,
              "end": 32000
            }
          ]
        },
        "role": "*"
      }
    ],
    "attributes": [
      {
        "name": "idc",
        "type": 3,
        "text": {
          "value": "idc-jh01"
        }
      },
      {
        "name": "id",
        "type": 3,
        "text": {
          "value": "hostid-ubuntu-s2"
        }
      },
      {
        "name": "hostid",
        "type": 3,
        "text": {
          "value": "hostid-ubuntu-s2"
        }
      },
      {
        "name": "stage",
        "type": 3,
        "text": {
          "value": "online"
        }
      }
    ]
  },
  "offerID": {
    "value": "8e9bd182-6693-493a-8edb-4c7e2cf90bbd-O30"
  },
  "time": "2016-08-11T16:29:03+08:00"
}

(2 more ...)
```


An Offer:
* Only contain resources from a single slave
* Resources associated with an offer will not be re-offered to this framework until either:
    - this framework has rejected those resources (`SchedulerDriver::lanuchTasks` will reject unused resources)
    - those resources have been rescinded (see `Scheduler::offerRescinded`).
* There are different types of resources in an agent.

So `simpleScheduler` will not receive any new offers since we do nothing on those offers we received.

If we reject all offers we received, those offer will be offered to us over and over again.

```
func (s *simpleScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	s.printOffers(offers)
	s.declineOffers(driver, offers)
}

func (s *simpleScheduler) declineOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	log.Printf("decline %d resource offers", len(offers))
	for _, offer := range offers {
		driver.DeclineOffer(offer.Id, defaultFilter)
	}
}
```

Logs:

```
{"level":"info","msg":"Received 3 resource offers","time":"2016-08-11T16:46:34+08:00"}
...
{"level":"info","msg":"decline 3 resource offers","time":"2016-08-11T16:46:34+08:00"}
{"level":"info","msg":"Received 3 resource offers","time":"2016-08-11T16:46:36+08:00"}
...
{"level":"info","msg":"decline 3 resource offers","time":"2016-08-11T16:46:36+08:00"}
{"level":"info","msg":"Received 3 resource offers","time":"2016-08-11T16:46:38+08:00"}
...
```
