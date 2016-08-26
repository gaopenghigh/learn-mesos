Run Docker Container
----

# Demo

According to Mesos document [Docker Containerizer](http://mesos.apache.org/documentation/latest/docker-containerizer/):

> To run a Docker image as a task, in TaskInfo one must set both the command and the container field as the Docker
Containerizer will use the accompanied command to launch the docker image. The ContainerInfo should have type Docker and
a DockerInfo that has the desired docker image.

The Docker Containerizer will translate Task/Executor `Launch` and `Destroy` calls to Docker CLI commands.

```
func (s *simpleScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	s.runCommandTasks(driver, offers, s.newDockerContainerTask)
}

...

func (s *simpleScheduler) newDockerContainerTask(cmd string, offer *mesos.Offer) *mesos.TaskInfo {
	taskCount = taskCount + 1
	task := &mesos.TaskInfo{
		TaskId: &mesos.TaskID{
			Value: proto.String(fmt.Sprintf("DockerContainerTask-%d", taskCount)),
		},
		Name: proto.String(fmt.Sprintf("DockerContainerTask-%d", taskCount)),
		SlaveId: offer.SlaveId,
		Resources: []*mesos.Resource{
			mesosutil.NewScalarResource("cpus", 0.5),
			mesosutil.NewScalarResource("mem", 100.0),
		},
		Command: &mesos.CommandInfo{
			Value: proto.String(cmd),
		},
		Container: &mesos.ContainerInfo{
			Type: mesos.ContainerInfo_DOCKER.Enum(),
			Docker: &mesos.ContainerInfo_DockerInfo{
				Image: proto.String("ugistry.ucloud.cn/library/centos:6.6"),
				Network: mesos.ContainerInfo_DockerInfo_BRIDGE.Enum(),
			},
		},
	}
	return task
}
```

Logs:

```
{
  "level": "info",
  "msg": "command task",
  "task": {
    "name": "DockerContainerTask-1",
    "task_id": {
      "value": "DockerContainerTask-1"
    },
    "slave_id": {
      "value": "f6f7b9a4-235d-4855-877f-a49e62294c2b-S0"
    },
    "resources": [
      {
        "name": "cpus",
        "type": 0,
        "scalar": {
          "value": 0.5
        }
      },
      {
        "name": "mem",
        "type": 0,
        "scalar": {
          "value": 100
        }
      }
    ],
    "command": {
      "value": "env \u0026\u0026 while true; do echo command running; sleep 3; done"
    },
    "container": {
      "type": 1,
      "docker": {
        "image": "ugistry.ucloud.cn/library/centos:6.6",
        "network": 2
      }
    }
  },
  "time": "2016-08-12T11:51:39+08:00"
}
```

From Mesos dashboard we can see the host that has this docker container.

Log into that host we can find out information of this container:
1. Name is `/mesos-f6f7b9a4-235d-4855-877f-a49e62294c2b-S0.f7f2b091-30e0-49c0-91e4-df965a8ae72d`
2. Network is bridge
3. Environments:
    ```
    "Env": [
       "MESOS_SANDBOX=/mnt/mesos/sandbox",
       "MESOS_CONTAINER_NAME=mesos-f6f7b9a4-235d-4855-877f-a49e62294c2b-S0.f7f2b091-30e0-49c0-91e4-df965a8ae72d"
    ],
    ```
4. Cmd:
    ```
    "Cmd": [
        "-c",
        "env \u0026\u0026 while true; do echo command running; sleep 3; done"
    ],
    ```
5. Mounts:
    ```
    "Mounts": [
        {
            "Source": "/home/jh/local/mesos-1.0.0/var/slave/slaves/f6f7b9a4-235d-4855-877f-a49e62294c2b-S0/frameworks/dbb6f685-5eea-4ad8-9c38-0e86d792cdf6-0005/executors/DockerContainerTask-1/runs/f7f2b091-30e0-49c0-91e4-df965a8ae72d",
            "Destination": "/mnt/mesos/sandbox",
            "Mode": "",
            "RW": true,
            "Propagation": "rprivate"
        }
    ],
    ```

# Describe a Docker Container

## Struct `ContainerInfo`

```
// *
// Describes a container configuration and allows extensible
// configurations for different container implementations.
type ContainerInfo struct {
	Type     *ContainerInfo_Type `protobuf:"varint,1,req,name=type,enum=mesosproto.ContainerInfo_Type" json:"type,omitempty"`
	Volumes  []*Volume           `protobuf:"bytes,2,rep,name=volumes" json:"volumes,omitempty"`
	Hostname *string             `protobuf:"bytes,4,opt,name=hostname" json:"hostname,omitempty"`
	// Only one of the following *Info messages should be set to match
	// the type.
	Docker *ContainerInfo_DockerInfo `protobuf:"bytes,3,opt,name=docker" json:"docker,omitempty"`
	Mesos  *ContainerInfo_MesosInfo  `protobuf:"bytes,5,opt,name=mesos" json:"mesos,omitempty"`
	// A list of network requests. A framework can request multiple IP addresses
	// for the container.
	NetworkInfos     []*NetworkInfo `protobuf:"bytes,7,rep,name=network_infos" json:"network_infos,omitempty"`
	XXX_unrecognized []byte         `json:"-"`
}
```

## Struct `ContainerInfo_DockerInfo`

```
type ContainerInfo_DockerInfo struct {
	// The docker image that is going to be passed to the registry.
	Image        *string                                 `protobuf:"bytes,1,req,name=image" json:"image,omitempty"`
	Network      *ContainerInfo_DockerInfo_Network       `protobuf:"varint,2,opt,name=network,enum=mesosproto.ContainerInfo_DockerInfo_Network,def=1" json:"network,omitempty"`
	PortMappings []*ContainerInfo_DockerInfo_PortMapping `protobuf:"bytes,3,rep,name=port_mappings" json:"port_mappings,omitempty"`
	Privileged   *bool                                   `protobuf:"varint,4,opt,name=privileged,def=0" json:"privileged,omitempty"`
	// Allowing arbitrary parameters to be passed to docker CLI.
	// Note that anything passed to this field is not guaranteed
	// to be supported moving forward, as we might move away from
	// the docker CLI.
	Parameters []*Parameter `protobuf:"bytes,5,rep,name=parameters" json:"parameters,omitempty"`
	// With this flag set to true, the docker containerizer will
	// pull the docker image from the registry even if the image
	// is already downloaded on the slave.
	ForcePullImage   *bool  `protobuf:"varint,6,opt,name=force_pull_image" json:"force_pull_image,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}
```
