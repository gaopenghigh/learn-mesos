package main

import (
	"strconv"
	"strings"

	"github.com/mesos/mesos-go/mesosproto"
)

func getPorts(offer *mesosproto.Offer) (ports []uint64) {
	for _, resource := range offer.Resources {
		if resource.GetName() == "ports" {
			for _, rang := range resource.GetRanges().GetRange() {
				for i := rang.GetBegin(); i <= rang.GetEnd(); i++ {
					ports = append(ports, i)
				}
			}
		}
	}
	return ports
}

func getContainerPorts(portMapsStr string) []int {
	ports := []int{}
	if len(portMapsStr) == 0 {
		return ports
	}
	for _, item := range strings.Split(portMapsStr, ",") {
		port, err := strconv.Atoi(item)
		checkErr(err)
		ports = append(ports, port)
	}
	return ports
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

// maxTasksForOffer computes how many tasks can be launched using a given offer
func maxTasksForOffer(offer *mesosproto.Offer) int {
	count := 0

	var cpus, mem float64

	for _, resource := range offer.Resources {
		switch resource.GetName() {
		case "cpus":
			cpus += *resource.GetScalar().Value
		case "mem":
			mem += *resource.GetScalar().Value
		}
	}

	for cpus >= taskCPUs && mem >= taskMem {
		count++
		cpus -= taskCPUs
		mem -= taskMem
	}

	return count
}
