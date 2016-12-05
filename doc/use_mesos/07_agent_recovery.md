Agent Recovery
----

In Mesos document [Agent Recovery](http://mesos.apache.org/documentation/latest/agent-recovery/):

> If the mesos-agent process on a host exits (perhaps due to a Mesos bug or because the operator kills the process
> while upgrading Mesos), any executors/tasks that were being managed by the mesos-agent process will continue to run.
> When mesos-agent is restarted, the operator can control how those old executors/tasks are handled:
> 1. By default, all the executors/tasks that were being managed by the old mesos-agent process are killed.
> 2. If a framework enabled checkpointing when it registered with the master, any executors belonging to that framework
>    can reconnect to the new mesos-agent process and continue running uninterrupted.

This is **INCORRECT**. Actually the test result is:

* If framework **disable checkpoint**(default):
    - When agent terminates, all running executors/tasks will be killed.
    - When new agent starts, it has no relationship with the older one.
* If framework **enable checkpoint**:
    - When agent terminates, all running executors/tasks continue to run
    - When new agent starts:
        1. If new agent starts within `agent_ping_timeout(default:15s) * max_agent_ping_timeouts(default:5)`, which is
           75s, all running executors/tasks continue to run.
        2. If new agent takes longer than `agent_ping_timeout(default:15s) * max_agent_ping_timeouts(default:5)` to
           re-register, the master will shuts down the agent, which in turn will shutdown any live executors/tasks.

Agent recovery works by having the agent checkpoint information (e.g., Task Info, Executor Info, Status Updates) about
the tasks and executors it is managing to local disk.

# Demo

## Checkpointing Disabled

Run a long-running task runs in a shell command task, like this:
```
./simple_scheduler -host 192.168.56.11 \
  -master 192.168.56.21:5050 \
  -cmd "env && ps aux && while true; do echo running; sleep 30; done"
```
Now process is running on one of the agents, in my case, agentID is `540dd450-6b47-42d1-affd-4f4b912a35f3-S2`:
```
# pstree -p
systemd(1)─┬─accounts-daemon(727)─┬─{gdbus}(793)
           │                      └─{gmain}(778)
           ├─mesos-agent(1859)─┬─mesos-executor(2362)─┬─sh(2383)───sleep(2429)
           │                   │                      ├─{mesos-executor}(2374)
           │                   │                      ├─{mesos-executor}(2375)
           │                   │                      ├─{mesos-executor}(2376)
           │                   │                      ├─{mesos-executor}(2377)
           │                   │                      ├─{mesos-executor}(2378)
           │                   │                      ├─{mesos-executor}(2379)
           │                   │                      ├─{mesos-executor}(2380)
           │                   │                      ├─{mesos-executor}(2381)
           │                   │                      └─{mesos-executor}(2382)
           │                   ├─{mesos-agent}(1860)
           │                   ├─{mesos-agent}(1861)
           │                   ├─{mesos-agent}(1862)
           │                   ├─{mesos-agent}(1863)
           │                   ├─{mesos-agent}(1864)
           │                   ├─{mesos-agent}(1865)
           │                   ├─{mesos-agent}(1866)
           │                   ├─{mesos-agent}(1867)
           │                   ├─{mesos-agent}(1868)
           │                   ├─{mesos-agent}(1983)
           │                   └─{mesos-agent}(1984)

...

# ps aux | grep -v grep | grep sleep
jh        2383  0.1  0.0   4508  1696 ?        Ss   10:29   0:00 sh -c env && ps aux && while true; do echo running; sleep 30; done
jh        2398  0.0  0.0   4380   676 ?        S    10:31   0:00 sleep 30
```

Use `kill` to stop mesos-agent.
Status of this task become `LOST`, both the `mesos-executor` process and the `sh` process are terminated.

Restart mesos-agent, the will not come back.
Actually, the is a totally new agent with a different agentID: `28ab248e-81b8-4f49-85c1-22c79f311cf8-S0`.

## Checkpoin Enabled

### Restart Agent After Timeout

Here _Timeout_ is `agent_ping_timeout(default:15s) * max_agent_ping_timeouts(default:5)`, which default is 75s.
 
Enable checkpointing in our framework:
```
...
	driver, err := sched.NewMesosSchedulerDriver(sched.DriverConfig{
		Master: *master,
		Framework: &mesos.FrameworkInfo{
			Name: proto.String("RENDLER"),
			User: proto.String(""),
			Role: proto.String(*role),
			Checkpoint: proto.Bool(*enableCheckPoint),
		},
		Scheduler:      scheduler,
		BindingAddress: net.ParseIP(*host),
	})
...
```

Start simpleScheduler like this:
```
./simple_scheduler -host 192.168.56.11 \
  -master 192.168.56.21:5050 \
  -enableCheckPoint \
  -cmd "env && ps aux && while true; do echo running; sleep 30; done"
```

Task started in agent with agentID=`28ab248e-81b8-4f49-85c1-22c79f311cf8-S0`:
```
# ps aux | grep -v grep | grep sleep
jh        2656  0.0  0.0   4508   856 ?        Ss   11:32   0:00 sh -c env && ps aux && while true; do echo running; sleep 30; done
jh        2666  0.0  0.0   4380   756 ?        S    11:34   0:00 sleep 30

# pstree -p
systemd(1)─┬─accounts-daemon(727)─┬─{gdbus}(793)
           │                      └─{gmain}(778)
           ├─mesos-agent(2596)─┬─mesos-executor(2634)─┬─sh(2656)───sleep(2666)
           │                   │                      ├─{mesos-executor}(2647)
           │                   │                      ├─{mesos-executor}(2648)
           │                   │                      ├─{mesos-executor}(2649)
           │                   │                      ├─{mesos-executor}(2650)
           │                   │                      ├─{mesos-executor}(2651)
           │                   │                      ├─{mesos-executor}(2652)
           │                   │                      ├─{mesos-executor}(2653)
           │                   │                      ├─{mesos-executor}(2654)
           │                   │                      └─{mesos-executor}(2655)
           │                   ├─{mesos-agent}(2597)
           │                   ├─{mesos-agent}(2598)
           │                   ├─{mesos-agent}(2599)
           │                   ├─{mesos-agent}(2600)
           │                   ├─{mesos-agent}(2601)
           │                   ├─{mesos-agent}(2602)
           │                   ├─{mesos-agent}(2603)
           │                   ├─{mesos-agent}(2604)
           │                   ├─{mesos-agent}(2605)
           │                   ├─{mesos-agent}(2616)
           │                   └─{mesos-agent}(2617)
...

```

Kill Agent, we will find both `mesos-executor` process and `sh` process are still alive:
```
# kill 2596

# pstree -p
systemd(1)─┬─accounts-daemon(727)─┬─{gdbus}(793)
           │                      └─{gmain}(778)
           ├─mesos-executor(2634)─┬─sh(2656)───sleep(2673)
           │                      ├─{mesos-executor}(2647)
           │                      ├─{mesos-executor}(2648)
           │                      ├─{mesos-executor}(2649)
           │                      ├─{mesos-executor}(2650)
           │                      ├─{mesos-executor}(2651)
           │                      ├─{mesos-executor}(2652)
           │                      ├─{mesos-executor}(2653)
           │                      ├─{mesos-executor}(2654)
           │                      └─{mesos-executor}(2655)
...
```

Task state is `RUNNING`, wait till timeout(75s), Task state became `LOST`, but he processes are still alive.

Now Start the agent, we can find that `mesos-executor` process and `sh` process will be killed, and the agent will exit.

In `stdout` of agent, we can find:
```
I0825 11:50:55.824367  2757 state.cpp:57] Recovering state from '/home/jh/local/mesos-1.0.0/var/slave/meta'
I0825 11:50:55.834473  2755 slave.cpp:4870] Recovering framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0002
I0825 11:50:55.834998  2755 slave.cpp:5798] Recovering executor 'ShellCommandTask-1' of framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0002
I0825 11:50:55.836971  2758 status_update_manager.cpp:200] Recovering status update manager
I0825 11:50:55.837483  2758 status_update_manager.cpp:208] Recovering executor 'ShellCommandTask-1' of framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0002
I0825 11:50:55.843927  2761 containerizer.cpp:522] Recovering containerizer
I0825 11:50:55.844440  2757 docker.cpp:775] Recovering Docker containers
I0825 11:50:55.848520  2761 containerizer.cpp:577] Recovering container '821fc9c8-e97a-4fe8-8ffb-a6cd26df73c1' for executor 'ShellCommandTask-1' of framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0002
...
I0825 11:50:55.877974  2756 metadata_manager.cpp:205] No images to load from disk. Docker provisioner image storage path '/tmp/mesos/store/docker/storedImages' does not exist
I0825 11:50:55.880913  2761 provisioner.cpp:253] Provisioner recovery complete
I0825 11:50:55.981637  2760 docker.cpp:870] Skipping recovery of executor 'ShellCommandTask-1' of framework '28ab248e-81b8-4f49-85c1-22c79f311cf8-0002' because its executor is not marked as docker and the docker container doesn't exist
I0825 11:50:55.992842  2760 slave.cpp:4722] Sending reconnect request to executor 'ShellCommandTask-1' of framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0002 at executor(1)@192.168.56.23:33912
I0825 11:50:56.002936  2760 slave.cpp:2998] Re-registering executor 'ShellCommandTask-1' of framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0002
...
I0825 11:50:58.004320  2757 slave.cpp:3151] Cleaning up un-reregistered executors
I0825 11:50:58.005480  2757 slave.cpp:4782] Finished recovery
...
I0825 11:50:58.347591  2759 slave.cpp:809] Agent asked to shut down by master@192.168.56.21:5050 because 'Agent attempted to re-register after removal'
I0825 11:50:58.349558  2759 slave.cpp:2218] Asked to shut down framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0002 by master@192.168.56.21:5050
I0825 11:50:58.350175  2759 slave.cpp:2243] Shutting down framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0002
I0825 11:50:58.350864  2759 slave.cpp:4407] Shutting down executor 'ShellCommandTask-1' of framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0002 at executor(1)@192.168.56.23:33912
...
I0825 11:50:59.543180  2754 slave.cpp:4082] Executor 'ShellCommandTask-1' of framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0002 has terminated with unknown status
I0825 11:50:59.543993  2754 slave.cpp:4193] Cleaning up executor 'ShellCommandTask-1' of framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0002 at executor(1)@192.168.56.23:33912
...
I0825 11:50:59.546716  2754 slave.cpp:4281] Cleaning up framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0002
I0825 11:50:59.548269  2754 slave.cpp:767] Agent terminating
...
```

### Restart Agent After Timeout

Here _Timeout_ is `agent_ping_timeout(default:15s) * max_agent_ping_timeouts(default:5)`, which default is 75s.

Run a shell command task on one of the agent, agentID=`540dd450-6b47-42d1-affd-4f4b912a35f3-S0`:
```
# ps aux | grep -v grep | grep sleep
jh        4996  0.0  0.0   4508  1696 ?        Ss   11:59   0:00 sh -c env && ps aux && while true; do echo running; sleep 30; done
jh        5027  0.0  0.0   4380   764 ?        S    12:00   0:00 sleep 30

# pstree -p
systemd(1)─┬─accounts-daemon(826)─┬─{gdbus}(860)
           │                      └─{gmain}(858)
           ├─mesos-agent(1883)─┬─mesos-executor(4974)─┬─sh(4996)───sleep(5031)
           │                   │                      ├─{mesos-executor}(4987)
           │                   │                      ├─{mesos-executor}(4988)
           │                   │                      ├─{mesos-executor}(4989)
           │                   │                      ├─{mesos-executor}(4990)
           │                   │                      ├─{mesos-executor}(4991)
           │                   │                      ├─{mesos-executor}(4992)
           │                   │                      ├─{mesos-executor}(4993)
           │                   │                      ├─{mesos-executor}(4994)
           │                   │                      └─{mesos-executor}(4995)
           │                   ├─{mesos-agent}(1884)
           │                   ├─{mesos-agent}(1885)
           │                   ├─{mesos-agent}(1886)
           │                   ├─{mesos-agent}(1887)
           │                   ├─{mesos-agent}(1888)
           │                   ├─{mesos-agent}(1889)
           │                   ├─{mesos-agent}(1890)
           │                   ├─{mesos-agent}(1891)
           │                   ├─{mesos-agent}(1892)
           │                   ├─{mesos-agent}(4785)
           │                   └─{mesos-agent}(4786)

```

Kill the agent, check process:
```
# kill 1883

# pstree -p
systemd(1)─┬─accounts-daemon(826)─┬─{gdbus}(860)
           │                      └─{gmain}(858)
           ├─mesos-executor(4974)─┬─sh(4996)───sleep(5057)
           │                      ├─{mesos-executor}(4987)
           │                      ├─{mesos-executor}(4988)
           │                      ├─{mesos-executor}(4989)
           │                      ├─{mesos-executor}(4990)
           │                      ├─{mesos-executor}(4991)
           │                      ├─{mesos-executor}(4992)
           │                      ├─{mesos-executor}(4993)
           │                      ├─{mesos-executor}(4994)
           │                      └─{mesos-executor}(4995)
...
```

Task state is `RUNNING`.

Start agent manually with 75s, check processes:
```
# pstree -p
systemd(1)─┬─accounts-daemon(826)─┬─{gdbus}(860)
           │                      └─{gmain}(858)
           ├─mesos-executor(4974)─┬─sh(4996)───sleep(5114)
           │                      ├─{mesos-executor}(4987)
           │                      ├─{mesos-executor}(4988)
           │                      ├─{mesos-executor}(4989)
           │                      ├─{mesos-executor}(4990)
           │                      ├─{mesos-executor}(4991)
           │                      ├─{mesos-executor}(4992)
           │                      ├─{mesos-executor}(4993)
           │                      ├─{mesos-executor}(4994)
           │                      └─{mesos-executor}(4995)
           ├─sshd(1249)───sshd(4811)─┬─bash(4866)───bash(5063)───mesos-agent(5064)─┬─{mesos-agent}(5065)
           │                         │                                             ├─{mesos-agent}(5066)
           │                         │                                             ├─{mesos-agent}(5067)
           │                         │                                             ├─{mesos-agent}(5068)
           │                         │                                             ├─{mesos-agent}(5069)
           │                         │                                             ├─{mesos-agent}(5070)
           │                         │                                             ├─{mesos-agent}(5071)
           │                         │                                             ├─{mesos-agent}(5072)
           │                         │                                             ├─{mesos-agent}(5073)
           │                         │                                             ├─{mesos-agent}(5085)
           │                         │                                             └─{mesos-agent}(5086)
           │                         └─bash(5092)───pstree(5115)
           ...

```

Task state is still `RUNNING`.
And the agentID of the newly started agent is the same: `540dd450-6b47-42d1-affd-4f4b912a35f3-S0`.

In `stdout` of agent, we can find:
```
I0825 12:12:50.825507  5072 slave.cpp:4870] Recovering framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0003
I0825 12:12:50.826236  5072 slave.cpp:5798] Recovering executor 'ShellCommandTask-1' of framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0003
I0825 12:12:50.828652  5069 status_update_manager.cpp:200] Recovering status update manager
I0825 12:12:50.828879  5069 status_update_manager.cpp:208] Recovering executor 'ShellCommandTask-1' of framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0003
I0825 12:12:50.832201  5071 containerizer.cpp:522] Recovering containerizer
I0825 12:12:50.832885  5071 containerizer.cpp:577] Recovering container '31e6f544-7675-410a-adfd-3dc4d5711636' for executor 'ShellCommandTask-1' of framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0003
I0825 12:12:50.832414  5068 docker.cpp:775] Recovering Docker containers
...
I0825 12:12:50.976593  5069 docker.cpp:870] Skipping recovery of executor 'ShellCommandTask-1' of framework '28ab248e-81b8-4f49-85c1-22c79f311cf8-0003' because its executor is not marked as docker and the docker container doesn't exist
I0825 12:12:50.984295  5067 slave.cpp:4722] Sending reconnect request to executor 'ShellCommandTask-1' of framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0003 at executor(1)@192.168.56.21:40359
I0825 12:12:50.997484  5071 slave.cpp:2998] Re-registering executor 'ShellCommandTask-1' of framework 28ab248e-81b8-4f49-85c1-22c79f311cf8-0003
...
I0825 12:12:52.996280  5069 slave.cpp:3151] Cleaning up un-reregistered executors
I0825 12:12:52.996791  5069 slave.cpp:4782] Finished recovery
...

```
