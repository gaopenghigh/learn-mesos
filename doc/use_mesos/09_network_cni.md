Network of Mesos Containerizer
----

The MesosContainerizer uses the `network/cni` isolator to implement the 
[Container Network Interface (CNI)](https://github.com/containernetworking/cni) to provide networking support for Mesos
containers.

# Container Network Interface(CNI)

From the view of network:
* _container_ is a Linux network namespace.
* _network_ refers to a group of entities that are uniquely addressable that can communicate amongst each other.

CNI defines a interface to can add/remove a container into/from a network.

## CNI Demo

Define 2 networks:
1. a bridge network named `mybridge0`, with interface `mybr0`
2. a loopback network

```
$ cat > /tmp/test-cni-loopback.conf <<EOF
{
    "type": "loopback"
}
EOF

$ cat > /tmp/test-cni-mybridge0.conf <<EOF
{
    "name": "mybridge0",
    "type": "bridge",
    "bridge": "mybr0",
    "isGateway": true,
    "ipMasq": true,
    "ipam": {
        "type": "host-local",
        "subnet": "10.23.0.0/24",
        "routes": [
            { "dst": "0.0.0.0/0" }
        ]
    }
}
EOF
```

Create a container(network namespace) named `mycontainer0`:
```
$ ip netns add mycontainer0

$ ip netns list
mycontainer0

$ ls -l /var/run/netns/
total 0
-r--r--r-- 1 root root 0 Aug 31 11:32 mycontainer0
```

Add container `mycontainer0` to loopback network:
```
$ cd /path/to/cni
$ export CNI_COMMAND=ADD
$ export CNI_CONTAINERID=mycontainer0
$ export CNI_NETNS=/var/run/netns/mycontainer0
$ export CNI_IFNAME=mynetlo
$ export CNI_PATH=`pwd`/bin
$ ./bin/loopback < /tmp/test-cni-loopback.conf
{
    "dns": {}
}
```

Check loopback network in container:

```
$ ip netns exec mycontainer0 ifconfig
lo        Link encap:Local Loopback  
          inet addr:127.0.0.1  Mask:255.0.0.0
          inet6 addr: ::1/128 Scope:Host
          UP LOOPBACK RUNNING  MTU:65536  Metric:1
          RX packets:0 errors:0 dropped:0 overruns:0 frame:0
          TX packets:0 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1 
          RX bytes:0 (0.0 B)  TX bytes:0 (0.0 B)
```

Add container `mycontainer0` to bridge network:
```
$ export CNI_COMMAND=ADD
$ export CNI_CONTAINERID=mycontainer0
$ export CNI_NETNS=/var/run/netns/mycontainer0
$ export CNI_IFNAME=eth0
$ export CNI_PATH=`pwd`/bin
$ ./bin/bridge < /tmp/test-cni-mybridge0.conf
{
    "ip4": {
        "ip": "10.23.0.3/24",
        "gateway": "10.23.0.1",
        "routes": [
            {
                "dst": "0.0.0.0/0"
            }
        ]
    },
    "dns": {}
}
```

Check bridge network:
```
$ ifconfig
...

lo        Link encap:Local Loopback  
          inet addr:127.0.0.1  Mask:255.0.0.0
          inet6 addr: ::1/128 Scope:Host
          UP LOOPBACK RUNNING  MTU:65536  Metric:1
          RX packets:161 errors:0 dropped:0 overruns:0 frame:0
          TX packets:161 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1 
          RX bytes:11928 (11.9 KB)  TX bytes:11928 (11.9 KB)

mybr0     Link encap:Ethernet  HWaddr 0a:58:0a:17:00:01  
          inet addr:10.23.0.1  Bcast:0.0.0.0  Mask:255.255.255.0
          inet6 addr: fe80::2897:87ff:fe90:4252/64 Scope:Link
          UP BROADCAST RUNNING MULTICAST  MTU:1500  Metric:1
          RX packets:8 errors:0 dropped:0 overruns:0 frame:0
          TX packets:8 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1000 
          RX bytes:536 (536.0 B)  TX bytes:648 (648.0 B)

veth381cf9da Link encap:Ethernet  HWaddr ce:50:b5:25:e2:48  
          inet6 addr: fe80::cc50:b5ff:fe25:e248/64 Scope:Link
          UP BROADCAST RUNNING MULTICAST  MTU:1500  Metric:1
          RX packets:8 errors:0 dropped:0 overruns:0 frame:0
          TX packets:16 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:0 
          RX bytes:648 (648.0 B)  TX bytes:1296 (1.2 KB)


$ ip netns exec mycontainer0 ifconfig
eth0      Link encap:Ethernet  HWaddr 0a:58:0a:17:00:03  
          inet addr:10.23.0.3  Bcast:0.0.0.0  Mask:255.255.255.0
          inet6 addr: fe80::18e9:a3ff:fe54:a5b1/64 Scope:Link
          UP BROADCAST RUNNING MULTICAST  MTU:1500  Metric:1
          RX packets:8 errors:0 dropped:0 overruns:0 frame:0
          TX packets:8 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:0 
          RX bytes:648 (648.0 B)  TX bytes:648 (648.0 B)

lo        Link encap:Local Loopback  
          inet addr:127.0.0.1  Mask:255.0.0.0
          inet6 addr: ::1/128 Scope:Host
          UP LOOPBACK RUNNING  MTU:65536  Metric:1
          RX packets:0 errors:0 dropped:0 overruns:0 frame:0
          TX packets:0 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1 
          RX bytes:0 (0.0 B)  TX bytes:0 (0.0 B)


$ ip netns exec mycontainer0 route -n
Kernel IP routing table
Destination     Gateway         Genmask         Flags Metric Ref    Use Iface
0.0.0.0         10.23.0.1       0.0.0.0         UG    0      0        0 eth0
10.23.0.0       0.0.0.0         255.255.255.0   U     0      0        0 eth0
```

Delete container from loopback and bridge network:
```
$ export CNI_COMMAND=DEL
$ export CNI_CONTAINERID=mycontainer0
$ export CNI_NETNS=/var/run/netns/mycontainer0
$ export CNI_IFNAME=eth0
$ export CNI_PATH=`pwd`/bin
$ ./bin/bridge < /tmp/test-cni-mybridge0.conf
$ ./bin/loopback < /tmp/test-cni-loopback.conf
$ ip netns exec mycontainer0 ifconfig
$
```

Delete container:
```
$ ip netns del mycontainer0
```
