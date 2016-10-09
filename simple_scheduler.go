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
	justPrintOffers  bool
	enableContainer  bool
	role             string
	containerType    string
	image            string
	network          string
	networkName      string
	exposePorts      []int
	reserveCPUs      float64
	reserveMem       float64
	alreadyReserved  bool
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
	if s.reserveMem > 0 || s.reserveCPUs > 0 {
		s.printOffers(offers)
		if !s.alreadyReserved {
			s.reserveResources(driver, offers[0])
		}
		s.declineOffers(driver, offers[1:])
		return
	}
	if s.justPrintOffers {
		s.printOffers(offers)
		return
	}
	if s.shellCmdQueue.Len() == 0 {
		s.declineOffers(driver, offers)
		return
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

func (s *demoScheduler) reserveResources(driver scheduler.SchedulerDriver, offer *mesosproto.Offer) {
	log.Infof("Reserving CPUs: %f, Mem: %f on offer %s", s.reserveCPUs, s.reserveMem, offer.Id)
	offerIDs := []*mesosproto.OfferID{offer.Id}
	cpuResource := mesosutil.NewScalarResource("cpus", s.reserveCPUs)
	cpuResource.Role = proto.String(s.role)
	memResource := mesosutil.NewScalarResource("mem", s.reserveMem)
	memResource.Role = proto.String(s.role)

	resources := []*mesosproto.Resource{cpuResource, memResource}
	log.WithFields(log.Fields{"resources": resources}).Info("reserve resources")
	operation := &mesosproto.Offer_Operation {
			Type: mesosproto.Offer_Operation_RESERVE.Enum(),
			Reserve: &mesosproto.Offer_Operation_Reserve{
				Resources: resources,
			},
	}
	log.WithFields(log.Fields{"operation": operation}).Info("reserve operation")
	operations := []*mesosproto.Offer_Operation{}
	operations = append(operations, operation)
	log.WithFields(log.Fields{
		"offer": offer,
		"operations": operations,
		"offerIDs": offerIDs,
	}).Info("reserve resource")
	status, err := driver.AcceptOffers(offerIDs, operations, defaultFilter)
	if err != nil {
		log.WithFields(log.Fields{"status": status, "err": err}).Error("reserve resource failed")
		panic(err)
	}
	log.WithFields(log.Fields{
		"status": status.String(),
		"offerIDs": offerIDs,
	}).Info("reserve resource success")
	s.alreadyReserved = true
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
	justPrintOffers := flag.Bool("justPrintOffers", false, "do nothing bug print offers")
	enableContainer := flag.Bool("enableContainer", false, "wether to use a container")
	enableCheckPoint := flag.Bool("enableCheckPoint", false, "wether to enable check point")
	containerType := flag.String("containerType", "docker",
		"type of container, useContainer need to be true, can be: mesosproto, docker, mesosprotoWithImage")
	image := flag.String("image", "", "image of container, useContainer need to be true")
	network := flag.String("network", "host", "docker containerizer: docker network type, host|bridge|none|...")
	networkName := flag.String("networkName", "", "mesos containerizer: name of CNI network to join")
	expose := flag.String("expose", "", "comma separated container ports e.g. 8080,8090,9000")
	reserveCPUs := flag.Float64("reserveCPUs", 0.0, "reserve cpus for role")
	reserveMem := flag.Float64("reserveMem", 0.0, "reserve mem for role")
	flag.Parse()

	if *enableContainer {
		if *image == "" {
			panic("image not specified")
		}
	}

	exposePorts := getContainerPorts(*expose)

	demoSche := &demoScheduler{
		enableContainer:  *enableContainer,
		justPrintOffers:  *justPrintOffers,
		enableCheckPoint: *enableCheckPoint,
		role:             *role,
		containerType:    *containerType,
		image:            *image,
		exposePorts:      exposePorts,
		reserveCPUs:      *reserveCPUs,
		reserveMem:      *reserveMem,
		network:          *network,
		networkName:      *networkName,
		alreadyReserved:  false,
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
