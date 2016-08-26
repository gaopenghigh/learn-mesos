Run Mesos Container with Image
----

Mesos Containerizer support container images like Docker and Appc.
And the image feature is not based on docker daemon.

Mesos use a component called _image provisioner_ to support images.
This is set via flag `--image_providers` when start agents.
Flag `--image_providers=docker,appc` allows both Docker and Appc images.

Two isolators are need to be turned on:
* `filesystem/linux`, because image support need to change filesystem root.
* `docker/runtime`, support runtime configuration specified in Docker images (e.g. Entrypoint/Cmd, Environments, etc)

So start agent with these flags(set by environment):
```
export MESOS_CONTAINERIZERS="mesos,docker"
export MESOS_ISOLATION="cgroups/cpu,cgroups/mem,namespaces/pid,filesystem/linux,docker/runtime"
export MESOS_IMAGE_PROVIDERS="docker"
```

And close Docker daemon.

# Demo

```
func (s *simpleScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	s.runCommandTasks(driver, offers, s.newMesosContainerWithDockerImageTask)
}

...


func (s *simpleScheduler) newMesosContainerWithDockerImageTask(cmd string, offer *mesos.Offer) *mesos.TaskInfo {
	taskCount = taskCount + 1
	task := &mesos.TaskInfo{
		TaskId: &mesos.TaskID{
			Value: proto.String(fmt.Sprintf("MesosContainerWithDockerImageTask-%d", taskCount)),
		},
		Name: proto.String(fmt.Sprintf("MesosContainerWithDockerImageTask-%d", taskCount)),
		SlaveId: offer.SlaveId,
		Resources: []*mesos.Resource{
			mesosutil.NewScalarResource("cpus", 1.0),
			mesosutil.NewScalarResource("mem", 200.0),
		},
		Command: &mesos.CommandInfo{
			Value: proto.String(cmd),
		},
		Container: &mesos.ContainerInfo{
			Type: mesos.ContainerInfo_MESOS.Enum(),
			Mesos: &mesos.ContainerInfo_MesosInfo{
				Image: &mesos.Image{
					Type: mesos.Image_DOCKER.Enum(),
					Docker: &mesos.Image_Docker{
						Name: proto.String("ugistry.ucloud.cn/library/centos:6.6"),
					},
				},
			},
		},
	}
	return task
}
```

Stdout of the task:

```
Received SUBSCRIBED event
Subscribed executor on 192.168.56.21
Received LAUNCH event
Starting task MesosContainerWithDockerImageTask-1
/home/jh/local/mesos-1.0.0/libexec/mesos/mesos-containerizer launch --command="{"shell":true,"value":"cat \/etc\/issue && ps aux && env && while true; do echo command running; sleep 3; done"}" --help="false" --rootfs="/home/jh/local/mesos-1.0.0/var/slave/provisioner/containers/168b386b-893f-478e-bbf1-209482231b88/backends/copy/rootfses/5eefc710-3eb1-4324-b895-61ddb85e8c94" --unshare_namespace_mnt="false" --user="jh" --working_directory="/mnt/mesos/sandbox"
Forked command at 20
Changing root to /home/jh/local/mesos-1.0.0/var/slave/provisioner/containers/168b386b-893f-478e-bbf1-209482231b88/backends/copy/rootfses/5eefc710-3eb1-4324-b895-61ddb85e8c94
CentOS release 6.6 (Final)
Kernel \r on an \m

USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
root         1 18.0  5.1 852568 105456 ?       Ssl  10:02   0:00 mesos-executor --launcher_dir=/home/jh/local/mesos-1.0.0/libexec/mesos --sandbox_directory=/mnt/mesos/sandbox --user=jh --rootfs=/home/jh/local/mesos-1.0.0/var/slave/provisioner/containers/168b386b-893f-478e-bbf1-209482231b88/backends/copy/rootfses/5eefc710-3eb1-4324-b895-61ddb85e8c94
1000        20  0.0  0.1  11360  2436 ?        Ss   10:02   0:00 sh -c cat /etc/issue && ps aux && env && while true; do echo command running; sleep 3; done
1000        22  0.0  0.0  13380  1840 ?        R    10:02   0:00 ps aux
LIBPROCESS_IP=192.168.56.21
MESOS_AGENT_ENDPOINT=192.168.56.21:5051
MESOS_DIRECTORY=/home/jh/local/mesos-1.0.0/var/slave/slaves/75558b63-46fa-421c-86fe-29cff15a3d5f-S0/frameworks/75558b63-46fa-421c-86fe-29cff15a3d5f-0001/executors/MesosContainerWithDockerImageTask-1/runs/168b386b-893f-478e-bbf1-209482231b88
MESOS_EXECUTOR_ID=MesosContainerWithDockerImageTask-1
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
PWD=/mnt/mesos/sandbox
MESOS_EXECUTOR_SHUTDOWN_GRACE_PERIOD=5secs
MESOS_NATIVE_JAVA_LIBRARY=/home/jh/local/mesos-1.0.0/lib/libmesos-1.0.0.so
MESOS_NATIVE_LIBRARY=/home/jh/local/mesos-1.0.0/lib/libmesos-1.0.0.so
MESOS_HTTP_COMMAND_EXECUTOR=0
MESOS_SLAVE_PID=slave(1)@192.168.56.21:5051
MESOS_FRAMEWORK_ID=75558b63-46fa-421c-86fe-29cff15a3d5f-0001
MESOS_CHECKPOINT=0
SHLVL=1
LIBPROCESS_PORT=0
MESOS_SLAVE_ID=75558b63-46fa-421c-86fe-29cff15a3d5f-S0
MESOS_SANDBOX=/mnt/mesos/sandbox
_=/usr/bin/env
command running
command running
...
```

Cgroups info:

```
# cat /proc/1979/cgroup 
11:devices:/user.slice/user-0.slice/session-2.scope
10:blkio:/user.slice/user-0.slice/session-2.scope
9:cpu,cpuacct:/mesos/168b386b-893f-478e-bbf1-209482231b88
8:perf_event:/
7:cpuset:/
6:memory:/mesos/168b386b-893f-478e-bbf1-209482231b88
5:hugetlb:/
4:freezer:/mesos/168b386b-893f-478e-bbf1-209482231b88
3:pids:/user.slice/user-0.slice/session-2.scope
2:net_cls,net_prio:/
1:name=systemd:/mesos_executors.slice


# cat /sys/fs/cgroup/memory/mesos/168b386b-893f-478e-bbf1-209482231b88/memory.limit_in_bytes 
243269632

```

Our host OS is Ubuntu, and the Image OS is CentOS, obviously the Docker image has been used without a Docker daemon.
The process is in it's own PID namespace, and resources are controlled by cgroups.

