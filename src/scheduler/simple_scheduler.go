package main

import (
	"container/list"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
	"github.com/mesos/mesos-go/scheduler"
)

const (
	containerTypeDocker         = "docker"
	containerTypeMesos          = "mesosproto"
	containerTypeMesosWithImage = "mesosprotoWithImage"
	taskCPUs                    = 0.1
	taskMem                     = 50.0
	shutdownTimeout             = time.Duration(3) * time.Second
	dockerNetworkBridge         = "bridge"
	dockerNetworkHost           = "host"
	dockerNetworkNone           = "none"
)

var (
	taskCount = 0
)

var (
	defaultFilter = &mesosproto.Filters{RefuseSeconds: proto.Float64(1)}
)

type demoScheduler struct {
	enableCheckPoint bool
	enableContainer  bool
	containerType    string
	image            string
	network          string
	exposePorts      []int
	shellCmdQueue    *list.List
	shutdown         chan struct{}
}

// handleSignal catch interrupt
func (s *demoScheduler) handleSignal(driver scheduler.SchedulerDriver) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	if sig != os.Interrupt {
		return
	}

	log.Println("RENDLER is shutting down")
	close(s.shutdown)

	select {
	case <-time.After(shutdownTimeout):
	}

	driver.Stop(false)
}

func (s *demoScheduler) newShellCommandTask(cmd string, offer *mesosproto.Offer) *mesosproto.TaskInfo {
	taskCount = taskCount + 1
	task := &mesosproto.TaskInfo{
		TaskId: &mesosproto.TaskID{
			Value: proto.String(fmt.Sprintf("ShellCommandTask-%d", taskCount)),
		},
		Name:    proto.String(fmt.Sprintf("ShellCommandTask-%d", taskCount)),
		SlaveId: offer.SlaveId,
		Resources: []*mesosproto.Resource{
			mesosutil.NewScalarResource("cpus", taskCPUs),
			mesosutil.NewScalarResource("mem", taskMem),
		},
		Command: &mesosproto.CommandInfo{
			Value: proto.String(cmd),
		},
	}
	return task
}

func (s *demoScheduler) newDockerContainerTask(cmd string, offer *mesosproto.Offer) *mesosproto.TaskInfo {
	taskCount = taskCount + 1

	var dockerNetwork *mesosproto.ContainerInfo_DockerInfo_Network
	switch s.network {
	case dockerNetworkNone:
		dockerNetwork = mesosproto.ContainerInfo_DockerInfo_NONE.Enum()
	case dockerNetworkBridge:
		dockerNetwork = mesosproto.ContainerInfo_DockerInfo_BRIDGE.Enum()
	case dockerNetworkHost:
		dockerNetwork = mesosproto.ContainerInfo_DockerInfo_HOST.Enum()
	default:
		panic("network type not supported")
	}

	var portMappings []*mesosproto.ContainerInfo_DockerInfo_PortMapping
	var usedPorts []int
	if len(s.exposePorts) > 0 {
		hostPorts := getPorts(offer)
		if len(s.exposePorts) > len(hostPorts) {
			panic("not enough ports")
		}
		for i, cp := range s.exposePorts {
			hp := hostPorts[i]
			pm := &mesosproto.ContainerInfo_DockerInfo_PortMapping{
				HostPort:      proto.Uint32(uint32(hp)),
				ContainerPort: proto.Uint32(uint32(cp)),
				Protocol:      proto.String("tcp"),
			}
			portMappings = append(portMappings, pm)
			usedPorts = append(usedPorts, int(hp))
		}
	}

	resources := []*mesosproto.Resource{
		mesosutil.NewScalarResource("cpus", taskCPUs),
		mesosutil.NewScalarResource("mem", taskMem),
	}
	for _, hostPort := range usedPorts {
		resources = append(resources,
			mesosutil.NewRangesResource("ports",
				[]*mesosproto.Value_Range{mesosutil.NewValueRange(uint64(hostPort), uint64(hostPort))}))
	}

	task := &mesosproto.TaskInfo{
		TaskId: &mesosproto.TaskID{
			Value: proto.String(fmt.Sprintf("DockerContainerTask-%d", taskCount)),
		},
		Name:      proto.String(fmt.Sprintf("DockerContainerTask-%d", taskCount)),
		SlaveId:   offer.SlaveId,
		Resources: resources,
		Command: &mesosproto.CommandInfo{
			Value: proto.String(cmd),
		},
		Container: &mesosproto.ContainerInfo{
			Type: mesosproto.ContainerInfo_DOCKER.Enum(),
			Docker: &mesosproto.ContainerInfo_DockerInfo{
				Image:        proto.String(s.image),
				Network:      dockerNetwork,
				PortMappings: portMappings,
			},
		},
	}
	return task
}

func (s *demoScheduler) newMesosContainerTask(cmd string, offer *mesosproto.Offer) *mesosproto.TaskInfo {
	taskCount = taskCount + 1
	task := &mesosproto.TaskInfo{
		TaskId: &mesosproto.TaskID{
			Value: proto.String(fmt.Sprintf("MesosContainerTask-%d", taskCount)),
		},
		Name:    proto.String(fmt.Sprintf("MesosContainerTask-%d", taskCount)),
		SlaveId: offer.SlaveId,
		Resources: []*mesosproto.Resource{
			mesosutil.NewScalarResource("cpus", taskCPUs),
			mesosutil.NewScalarResource("mem", taskMem),
		},
		Command: &mesosproto.CommandInfo{
			Value: proto.String(cmd),
		},
		Container: &mesosproto.ContainerInfo{
			Type:  mesosproto.ContainerInfo_MESOS.Enum(),
			Mesos: &mesosproto.ContainerInfo_MesosInfo{},
		},
	}
	return task
}

func (s *demoScheduler) newMesosContainerWithDockerImageTask(cmd string, offer *mesosproto.Offer) *mesosproto.TaskInfo {
	taskCount = taskCount + 1
	task := &mesosproto.TaskInfo{
		TaskId: &mesosproto.TaskID{
			Value: proto.String(fmt.Sprintf("MesosContainerWithDockerImageTask-%d", taskCount)),
		},
		Name:    proto.String(fmt.Sprintf("MesosContainerWithDockerImageTask-%d", taskCount)),
		SlaveId: offer.SlaveId,
		Resources: []*mesosproto.Resource{
			mesosutil.NewScalarResource("cpus", taskCPUs),
			mesosutil.NewScalarResource("mem", taskMem),
		},
		Command: &mesosproto.CommandInfo{
			Value: proto.String(cmd),
		},
		Container: &mesosproto.ContainerInfo{
			Type: mesosproto.ContainerInfo_MESOS.Enum(),
			Mesos: &mesosproto.ContainerInfo_MesosInfo{
				Image: &mesosproto.Image{
					Type: mesosproto.Image_DOCKER.Enum(),
					Docker: &mesosproto.Image_Docker{
						Name: proto.String(s.image),
					},
				},
			},
		},
	}
	return task
}

func (s *demoScheduler) Registered(
	_ scheduler.SchedulerDriver,
	frameworkID *mesosproto.FrameworkID,
	masterInfo *mesosproto.MasterInfo) {
	log.WithFields(log.Fields{"frameworkID": frameworkID, "masterInfo": masterInfo}).Info("framework registered")
}

func (s *demoScheduler) Reregistered(_ scheduler.SchedulerDriver, masterInfo *mesosproto.MasterInfo) {
	log.WithFields(log.Fields{"masterInfo": masterInfo}).Info("framework re-registered")
}

func (s *demoScheduler) Disconnected(scheduler.SchedulerDriver) {
	log.Println("Framework disconnected with master")
}

func (s *demoScheduler) ResourceOffers(driver scheduler.SchedulerDriver, offers []*mesosproto.Offer) {
	// s.printOffers(offers)
	if s.shellCmdQueue.Len() == 0 {
		s.declineOffers(driver, offers)
	}

	if !s.enableContainer {
		s.runCommandTasks(driver, offers, s.newShellCommandTask)
		return
	}

	if s.containerType == containerTypeDocker {
		s.runCommandTasks(driver, offers, s.newDockerContainerTask)
		return
	}
	if s.containerType == containerTypeMesos {
		s.runCommandTasks(driver, offers, s.newMesosContainerTask)
		return
	}
	if s.containerType == containerTypeMesosWithImage {
		s.runCommandTasks(driver, offers, s.newMesosContainerWithDockerImageTask)
		return
	}
	panic("unsupported container type")
}

func (s *demoScheduler) printOffers(offers []*mesosproto.Offer) {
	log.Infof("Received %d resource offers", len(offers))
	for _, offer := range offers {
		log.WithFields(log.Fields{"offerID": offer.Id, "offer": offer}).Info("offer")
	}
}

func (s *demoScheduler) declineOffers(driver scheduler.SchedulerDriver, offers []*mesosproto.Offer) {
	log.Debugf("decline %d resource offers", len(offers))
	for _, offer := range offers {
		driver.DeclineOffer(offer.Id, defaultFilter)
	}
}

func (s *demoScheduler) runCommandTasks(
	driver scheduler.SchedulerDriver,
	offers []*mesosproto.Offer,
	taskFactory func(cmd string, offer *mesosproto.Offer) *mesosproto.TaskInfo) {

	log.Debugf("Received %d resource offers", len(offers))
	for _, offer := range offers {
		log.WithFields(log.Fields{"offer": offer.String()}).Debugf("offer")
		select {
		case <-s.shutdown:
			log.Println("Shutting down: declining offer on [", offer.Hostname, "]")
			driver.DeclineOffer(offer.Id, defaultFilter)
			continue
		default:
		}

		tasks := []*mesosproto.TaskInfo{}
		tasksToLaunch := maxTasksForOffer(offer)
		log.Debugf("tasksToLaunch = %d\n", tasksToLaunch)
		for tasksToLaunch > 0 {
			if s.shellCmdQueue.Front() != nil {
				cmd := s.shellCmdQueue.Front().Value.(string)
				s.shellCmdQueue.Remove(s.shellCmdQueue.Front())
				task := taskFactory(cmd, offer)
				log.WithFields(log.Fields{"task": task}).Info("command task")
				tasks = append(tasks, task)
				tasksToLaunch--
			}
			if s.shellCmdQueue.Front() == nil {
				break
			}
		}

		if len(tasks) == 0 {
			driver.DeclineOffer(offer.Id, defaultFilter)
		} else {
			driver.LaunchTasks([]*mesosproto.OfferID{offer.Id}, tasks, defaultFilter)
		}
	}
}

func (s *demoScheduler) StatusUpdate(driver scheduler.SchedulerDriver, status *mesosproto.TaskStatus) {
	reason := ""
	if status.Reason != nil {
		reason = status.Reason.String()
	}
	log.WithFields(log.Fields{
		"taskID":          *status.TaskId.Value,
		"status":          status.State.String(),
		"reason":          reason,
		"source":          status.Source.String(),
		"containerStatus": status.ContainerStatus,
	}).Info("received task status")
}

func (s *demoScheduler) FrameworkMessage(
	driver scheduler.SchedulerDriver,
	executorID *mesosproto.ExecutorID,
	slaveID *mesosproto.SlaveID,
	message string) {

	log.WithFields(log.Fields{
		"executorID": executorID,
		"slaveID":    slaveID,
		"message":    message,
	}).Info("got a framework message")
}

func (s *demoScheduler) OfferRescinded(_ scheduler.SchedulerDriver, offerID *mesosproto.OfferID) {
	log.Printf("Offer %s rescinded", offerID)
}
func (s *demoScheduler) SlaveLost(_ scheduler.SchedulerDriver, slaveID *mesosproto.SlaveID) {
	log.Printf("Slave %s lost", slaveID)
}
func (s *demoScheduler) ExecutorLost(_ scheduler.SchedulerDriver, executorID *mesosproto.ExecutorID, slaveID *mesosproto.SlaveID,
	status int) {
	log.Printf("Executor %s on slave %s was lost", executorID, slaveID)
}

func (s *demoScheduler) Error(_ scheduler.SchedulerDriver, err string) {
	log.Printf("Receiving an error: %s", err)
}

func init() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.JSONFormatter{})
}

func main() {
	master := flag.String("master", "127.0.1.1:5050", "Location of leading Mesos master")
	host := flag.String("host", "127.0.0.1", "ip address which the framework bind")
	role := flag.String("role", "*", "framework role")
	taskNum := flag.Int("taskNum", 1, "number of tasks")
	cmd := flag.String("cmd", "while true; do echo command running; sleep 10; done", "shell command")
	enableContainer := flag.Bool("enableContainer", false, "wether to use a container")
	enableCheckPoint := flag.Bool("enableCheckPoint", false, "wether to enable check point")
	containerType := flag.String("containerType", "docker",
		"type of container, useContainer need to be true, can be: mesosproto, docker, mesosprotoWithImage")
	image := flag.String("image", "", "image of container, useContainer need to be true")
	network := flag.String("network", "host", "docker network type, host|bridge|none|...")
	expose := flag.String("expose", "", "comma separated container ports e.g. 8080,8090,9000")
	flag.Parse()

	if *enableContainer {
		if *image == "" {
			panic("image not specified")
		}
	}

	exposePorts := getContainerPorts(*expose)

	demoSche := &demoScheduler{
		enableContainer:  *enableContainer,
		enableCheckPoint: *enableCheckPoint,
		containerType:    *containerType,
		image:            *image,
		exposePorts:      exposePorts,
		network:          *network,
		shutdown:         make(chan struct{}),
		shellCmdQueue:    list.New(),
	}
	for i := 0; i < *taskNum; i++ {
		demoSche.shellCmdQueue.PushBack(*cmd)
	}

	driver, err := scheduler.NewMesosSchedulerDriver(scheduler.DriverConfig{
		Master: *master,
		Framework: &mesosproto.FrameworkInfo{
			Name:       proto.String("RENDLER"),
			User:       proto.String(""),
			Role:       proto.String(*role),
			Checkpoint: proto.Bool(*enableCheckPoint),
		},
		Scheduler:      demoSche,
		BindingAddress: net.ParseIP(*host),
	})
	if err != nil {
		log.Printf("Unable to create scheduler driver: %s", err)
		return
	}

	go demoSche.handleSignal(driver)

	if status, err := driver.Run(); err != nil {
		log.Printf("Framework stopped with status %s and error: %s\n", status.String(), err.Error())
	}
	log.Println("Exiting...")
}
