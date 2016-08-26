Run a Mesos Container
----

# Demo

Use Mesos Containerizer to run a container.

```
func (s *simpleScheduler) newMesosContainerTask(cmd string, offer *mesos.Offer) *mesos.TaskInfo {
	taskCount = taskCount + 1
	task := &mesos.TaskInfo{
		TaskId: &mesos.TaskID{
			Value: proto.String(fmt.Sprintf("MesosContainerTask-%d", taskCount)),
		},
		Name: proto.String(fmt.Sprintf("MesosContainerTask-%d", taskCount)),
		SlaveId: offer.SlaveId,
		Resources: []*mesos.Resource{
			mesosutil.NewScalarResource("cpus", 0.5),
			mesosutil.NewScalarResource("mem", 100.0),
		},
		Command: &mesos.CommandInfo{
			Value: proto.String(cmd),
		},
		Container: &mesos.ContainerInfo{
			Type: mesos.ContainerInfo_MESOS.Enum(),
			Mesos: &mesos.ContainerInfo_MesosInfo{
			},
		},
	}
	return task
}

...


func (s *simpleScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	s.runCommandTasks(driver, offers, s.newMesosContainerTask)
}
```

We can find our process(pid = 6812) like:

```
# pstree -p
systemd(1)─┬─accounts-daemon(866)─┬─{gdbus}(910)
           │                      └─{gmain}(891)
           | ....
           ├─mesos-agent(2879)─┬─mesos-executor(6801)─┬─sh(6812)───sleep(6882)
           │                   │                      ├─{mesos-executor}(6803)
           │                   │                      ├─{mesos-executor}(6804)
...
```

And our process was added to a `freezer` cgroup named `/mesos/9a3f7dcb-ab56-4b7f-a72e-47b60653d10d`:

```
# cat /proc/6812/cgroup 
11:blkio:/user.slice/user-0.slice/session-13.scope
10:cpu,cpuacct:/user.slice/user-0.slice/session-13.scope
9:cpuset:/
8:freezer:/mesos/9a3f7dcb-ab56-4b7f-a72e-47b60653d10d
7:devices:/user.slice/user-0.slice/session-13.scope
6:perf_event:/
5:pids:/user.slice/user-0.slice/session-13.scope
4:hugetlb:/
3:memory:/user.slice/user-0.slice/session-13.scope
2:net_cls,net_prio:/
1:name=systemd:/mesos_executors.slice
```

Since we are using the default flag `--isolation` to start Mesos agent, default value is `posix/cpu,posix/mem`.
According to an [Answer from StackOverflow](https://stackoverflow.com/questions/34608691/what-is-posix-isolation/34767231#34767231):

> Mesos uses isolators in its containerizer to isolate the resource usage of each started task. The posix isolators
provide separation of task as standard Unix processes. Each Mesos task is running under mesos-executor process, which 
makes sure that the task is running. These don't really perform any actual isolation but only report the current
resource usage of running tasks.

Restart the agents with flag `--isolation=cgroups/cpu,cgroups/mem,namespaces/pid`, and run Mesos container again, we can
find the process has been added into several cgroups:
```
# cat /proc/12620/cgroup 
11:blkio:/user.slice/user-0.slice/session-38.scope
10:freezer:/mesos/3e84fc0b-208f-4108-8a84-978b82ddd5c2
9:devices:/user.slice/user-0.slice/session-38.scope
8:perf_event:/
7:cpuset:/
6:pids:/user.slice/user-0.slice/session-38.scope
5:cpu,cpuacct:/mesos/3e84fc0b-208f-4108-8a84-978b82ddd5c2
4:hugetlb:/
3:memory:/mesos/3e84fc0b-208f-4108-8a84-978b82ddd5c2
2:net_cls,net_prio:/
1:name=systemd:/mesos_executors.slice


# cat /sys/fs/cgroup/memory/mesos/3e84fc0b-208f-4108-8a84-978b82ddd5c2/memory.limit_in_bytes 
138412032

# cat /sys/fs/cgroup/cpu,cpuacct/mesos/3e84fc0b-208f-4108-8a84-978b82ddd5c2/cpu.shares 
614
```

And our process and `mesos-executor` process are in a PID namespace:
```
USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
jh           1  0.0 10.3 844120 104904 ?       Ssl  16:13   0:00 mesos-executor --launcher_dir=/home/jh/local/mesos-1.0.0/libexec/mesos
jh          17  0.0  0.0   4508   752 ?        Ss   16:13   0:00 sh -c cat /etc/issue && ps aux && env && while true; do echo command running; sleep 3; done
jh          19  0.0  0.2  34424  2944 ?        R    16:13   0:00 ps aux
```

Both our process and the `mesos-executor` process are in these cgroups.

Resource of out container is cpu=0.5, mem=100M.
While the cgroups is `memory.limit_in_bytes` = 132M and `cpu.shares` = 614.

Start a new Container with cpu=1.0, mem=200M, we can find `memory.limit_in_byts` = 232M, and `cpu.shares` = 1126.

**Looks like add 32M memory and 102 cpu shares for `mesos-executor`.**


# Describe a Mesos Container

## Struct `ContainerInfo_MesosInfo`

```
type ContainerInfo_MesosInfo struct {
	Image            *Image `protobuf:"bytes,1,opt,name=image" json:"image,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}
```

Actually we did not use any `Image` in our demo.
