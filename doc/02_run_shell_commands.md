Run Shell Commands
----

# Demo

I will run a shell command task that use 1 cpu, 100m memory.


```
func (s *simpleScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	s.runCommandTasks(driver, offers, s.newShellCommandTask)
}

...

func (s *simpleScheduler) newShellCommandTask(cmd string, offer *mesos.Offer) *mesos.TaskInfo {
	taskCount = taskCount + 1
	task := &mesos.TaskInfo{
		TaskId: &mesos.TaskID{
			Value: proto.String(fmt.Sprintf("ShellCommandTask-%d", taskCount)),
		},
		Name: proto.String(fmt.Sprintf("ShellCommandTask-%d", taskCount)),
		SlaveId: offer.SlaveId,
		Resources: []*mesos.Resource{
			mesosutil.NewScalarResource("cpus", 1.0),
			mesosutil.NewScalarResource("mem", 100.0),
		},
		Command: &mesos.CommandInfo{
			Value: proto.String(cmd),
		},
	}
	return task
}
```

There is **no isolation** here since we haven't use any isolator.
So you can set resources like cpu=0.01 and mem=0.01, and the program will run successful.


# Describe a shell command Task

## Struct `TaskInfo`

Mesos use a `TaskInfo` to describe a task. Golang structure of `TaskInfo` is:

```
// *
// Describes a task. Passed from the scheduler all the way to an
// executor (see SchedulerDriver::launchTasks and
// Executor::launchTask). Either ExecutorInfo or CommandInfo should be set.
// A different executor can be used to launch this task, and subsequent tasks
// meant for the same executor can reuse the same ExecutorInfo struct.
type TaskInfo struct {
	Name      *string       `protobuf:"bytes,1,req,name=name" json:"name,omitempty"`
	TaskId    *TaskID       `protobuf:"bytes,2,req,name=task_id" json:"task_id,omitempty"`
	SlaveId   *SlaveID      `protobuf:"bytes,3,req,name=slave_id" json:"slave_id,omitempty"`
	Resources []*Resource   `protobuf:"bytes,4,rep,name=resources" json:"resources,omitempty"`
	Executor  *ExecutorInfo `protobuf:"bytes,5,opt,name=executor" json:"executor,omitempty"`
	Command   *CommandInfo  `protobuf:"bytes,7,opt,name=command" json:"command,omitempty"`
	// Task provided with a container will launch the container as part
	// of this task paired with the task's CommandInfo.
	Container *ContainerInfo `protobuf:"bytes,9,opt,name=container" json:"container,omitempty"`
	Data      []byte         `protobuf:"bytes,6,opt,name=data" json:"data,omitempty"`
	// A health check for the task (currently in *alpha* and initial
	// support will only be for TaskInfo's that have a CommandInfo).
	HealthCheck *HealthCheck `protobuf:"bytes,8,opt,name=health_check" json:"health_check,omitempty"`
	// Labels are free-form key value pairs which are exposed through
	// master and slave endpoints. Labels will not be interpreted or
	// acted upon by Mesos itself. As opposed to the data field, labels
	// will be kept in memory on master and slave processes. Therefore,
	// labels should be used to tag tasks with light-weight meta-data.
	Labels *Labels `protobuf:"bytes,10,opt,name=labels" json:"labels,omitempty"`
	// Service discovery information for the task. It is not interpreted
	// or acted upon by Mesos. It is up to a service discovery system
	// to use this information as needed and to handle tasks without
	// service discovery information.
	Discovery        *DiscoveryInfo `protobuf:"bytes,11,opt,name=discovery" json:"discovery,omitempty"`
	XXX_unrecognized []byte         `json:"-"`
}
```

Either `ExecutorInfo` or `CommandInfo` should be set.
Set `ExecutorInfo` if using an executor, if not, set `CommandInfo`.
I left `TaskInfo.Executor` and `TaskInfo.Container` to be `nil` since we haven't use a Executor or a Container.


## Struct `CommandInfo`

Structure `CommandInfo` describe the actual command.

```
// *
// Describes a command, executed via: '/bin/sh -c value'. Any URIs specified
// are fetched before executing the command.  If the executable field for an
// uri is set, executable file permission is set on the downloaded file.
// Otherwise, if the downloaded file has a recognized archive extension
// (currently [compressed] tar and zip) it is extracted into the executor's
// working directory. This extraction can be disabled by setting `extract` to
// false. In addition, any environment variables are set before executing
// the command (so they can be used to "parameterize" your command).
type CommandInfo struct {
	// NOTE: MesosContainerizer does currently not support this
	// attribute and tasks supplying a 'container' will fail.
	Container   *CommandInfo_ContainerInfo `protobuf:"bytes,4,opt,name=container" json:"container,omitempty"`
	Uris        []*CommandInfo_URI         `protobuf:"bytes,1,rep,name=uris" json:"uris,omitempty"`
	Environment *Environment               `protobuf:"bytes,2,opt,name=environment" json:"environment,omitempty"`
	// There are two ways to specify the command:
	// 1) If 'shell == true', the command will be launched via shell
	// 		(i.e., /bin/sh -c 'value'). The 'value' specified will be
	// 		treated as the shell command. The 'arguments' will be ignored.
	// 2) If 'shell == false', the command will be launched by passing
	// 		arguments to an executable. The 'value' specified will be
	// 		treated as the filename of the executable. The 'arguments'
	// 		will be treated as the arguments to the executable. This is
	// 		similar to how POSIX exec families launch processes (i.e.,
	// 		execlp(value, arguments(0), arguments(1), ...)).
	// NOTE: The field 'value' is changed from 'required' to 'optional'
	// in 0.20.0. It will only cause issues if a new framework is
	// connecting to an old master.
	Shell     *bool    `protobuf:"varint,6,opt,name=shell,def=1" json:"shell,omitempty"`
	Value     *string  `protobuf:"bytes,3,opt,name=value" json:"value,omitempty"`
	Arguments []string `protobuf:"bytes,7,rep,name=arguments" json:"arguments,omitempty"`
	// Enables executor and tasks to run as a specific user. If the user
	// field is present both in FrameworkInfo and here, the CommandInfo
	// user value takes precedence.
	User             *string `protobuf:"bytes,5,opt,name=user" json:"user,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}
```

Uris and Environments can be set here.