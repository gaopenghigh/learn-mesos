Docker Network in Mesos
----

Mesos support all kinds of Docker network types: `HOST`, `BRIDGE`, `HOST`, `NONE` and `USER`. 

## HOST Network

Start a Docker container task that use HOST network:
```
./simple_scheduler \
    -host=192.168.56.11 \
    -master 192.168.56.21:5050 \
    -enableContainer \
    -containerType=docker \
    -image "ugistry.ucloud.cn/library/centos:6.6" \
    -network host \
    -cmd "ifconfig && sleep 600"
```

Logs:
```
...
{
    "time":"2016-08-26T15:43:15+08:00"
    "level":"info",
    "taskID":"DockerContainerTask-1",
    "containerStatus":{"network_infos":[{"ip_addresses":[{"ip_address":"192.168.56.22"}]}]},
    "msg":"received task status",
    "reason":"",
    "source":"SOURCE_EXECUTOR",
    "status":"TASK_RUNNING",
}
...
```

Check the Docker container

```
root@ubuntu-s2:~# docker ps
CONTAINER ID        IMAGE                                  COMMAND                  CREATED             STATUS              PORTS               NAMES
d89789a87674        ugistry.ucloud.cn/library/centos:6.6   "/bin/sh -c 'ifconfig"   15 seconds ago      Up 15 seconds                           mesos-3357c1dc-7923-4afb-a770-fc5ed7d1a92b-S1.0ce834db-cbbc-4eb2-bf70-ea88ee42eda6

root@ubuntu-s2:~# docker logs d89789a87674
docker0   Link encap:Ethernet  HWaddr 02:42:0E:8E:8C:A7
          inet addr:172.17.0.1  Bcast:0.0.0.0  Mask:255.255.0.0
          UP BROADCAST MULTICAST  MTU:1500  Metric:1
          RX packets:0 errors:0 dropped:0 overruns:0 frame:0
          TX packets:0 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:0
          RX bytes:0 (0.0 b)  TX bytes:0 (0.0 b)

enp0s3    Link encap:Ethernet  HWaddr 08:00:27:5D:92:83
          inet addr:10.0.2.15  Bcast:10.0.2.255  Mask:255.255.255.0
          inet6 addr: fe80::a00:27ff:fe5d:9283/64 Scope:Link
          UP BROADCAST RUNNING MULTICAST  MTU:1500  Metric:1
          RX packets:20 errors:0 dropped:0 overruns:0 frame:0
          TX packets:29 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1000
          RX bytes:2236 (2.1 KiB)  TX bytes:2676 (2.6 KiB)

enp0s8    Link encap:Ethernet  HWaddr 08:00:27:6B:FE:B2
          inet addr:192.168.56.22  Bcast:192.168.56.255  Mask:255.255.255.0
          inet6 addr: fe80::a00:27ff:fe6b:feb2/64 Scope:Link
          UP BROADCAST RUNNING MULTICAST  MTU:1500  Metric:1
          RX packets:2414 errors:0 dropped:0 overruns:0 frame:0
          TX packets:2931 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1000
          RX bytes:355779 (347.4 KiB)  TX bytes:244530 (238.7 KiB)

lo        Link encap:Local Loopback
          inet addr:127.0.0.1  Mask:255.0.0.0
          inet6 addr: ::1/128 Scope:Host
          UP LOOPBACK RUNNING  MTU:65536  Metric:1
          RX packets:478 errors:0 dropped:0 overruns:0 frame:0
          TX packets:478 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1
          RX bytes:33671 (32.8 KiB)  TX bytes:33671 (32.8 KiB)

```

## BRIDGE Network

In most cases, port map is used when using BRIDGE network, port map can be set in `TaskInfo.ContainerInfo.Docker`.

Two things to remember:
1. Host port must in the port range resource provided by mesos agent.
2. Host port must set in the resources of the task.

```
./simple_scheduler \
    -enableContainer \
    -containerType=docker \
    -host=192.168.56.11 \
    -image "ugistry.ucloud.cn/library/centos:6.6" \
    -master=192.168.56.23:5050 \
    -network=bridge \
    -expose 8888 \
    -cmd "ifconfig && sleep 3000"
    
...
{
    "level":"info",
    "msg":"command task",
    "task":{
        "name":"DockerContainerTask-1",
        "task_id":{"value":"DockerContainerTask-1"},
        "slave_id":{"value":"3357c1dc-7923-4afb-a770-fc5ed7d1a92b-S0"},
        "resources":[
            {"name":"cpus","type":0,"scalar":{"value":0.1}},
            {"name":"mem","type":0,"scalar":{"value":50}},
            {"name":"ports","type":1,"ranges":{"range":[{"begin":31000,"end":31000}]}}
        ],
        "command":{"value":"ifconfig \u0026\u0026 sleep 3000"},
        "container":{
            "type":1,
            "docker":{
                "image":"ugistry.ucloud.cn/library/centos:6.6",
                "network":2,
                "port_mappings":[{"host_port":31000,"container_port":8888,"protocol":"tcp"}]
            }
        }
    },
    "time":"2016-08-26T18:06:58+08:00"
}
{
    "containerStatus":{"network_infos":[{"ip_addresses":[{"ip_address":"172.17.0.2"}]}]},
    "level":"info",
    "msg":"received task status",
    "reason":"",
    "source":"SOURCE_EXECUTOR",
    "status":"TASK_RUNNING",
    "taskID":"DockerContainerTask-1",
    "time":"2016-08-26T18:06:59+08:00"
}
```

Check:
```
# docker ps
CONTAINER ID        IMAGE                                  COMMAND                  CREATED             STATUS              PORTS                     NAMES
1b5172dc6466        ugistry.ucloud.cn/library/centos:6.6   "/bin/sh -c 'ifconfig"   2 minutes ago       Up 2 minutes        0.0.0.0:31000->8888/tcp   mesos-3357c1dc-7923-4afb-a770-fc5ed7d1a92b-S0.09c73ede-2d3c-4d05-9c4d-4d579621725f
```

## NONE Network

Command:
```
./simple_scheduler \
    -host=192.168.56.11 \
    -master 192.168.56.21:5050 \
    -enableContainer \
    -containerType=docker \
    -image "ugistry.ucloud.cn/library/centos:6.6" \
    -network none \
    -cmd "ifconfig && sleep 600"
```

Check:
```
# docker ps
CONTAINER ID        IMAGE                                  COMMAND                  CREATED              STATUS              PORTS               NAMES
8e46673ee4ed        ugistry.ucloud.cn/library/centos:6.6   "/bin/sh -c 'ifconfig"   About a minute ago   Up About a minute                       mesos-3357c1dc-7923-4afb-a770-fc5ed7d1a92b-S0.7e7c20e3-ce2b-4554-ae8a-0560b85cc0bd
root@ubuntu-s1:/home/jh/local/mesos-1.0.0# docker logs 8e46673ee4ed
lo        Link encap:Local Loopback
          inet addr:127.0.0.1  Mask:255.0.0.0
          inet6 addr: ::1/128 Scope:Host
          UP LOOPBACK RUNNING  MTU:65536  Metric:1
          RX packets:0 errors:0 dropped:0 overruns:0 frame:0
          TX packets:0 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1
          RX bytes:0 (0.0 b)  TX bytes:0 (0.0 b)

```
