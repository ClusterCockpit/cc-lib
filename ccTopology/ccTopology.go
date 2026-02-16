// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccTopology

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

/*
#cgo LDFLAGS: -lhwloc
#cgo linux LDFLAGS: -Wl,--unresolved-symbols=ignore-in-object-files -lhwloc
#cgo CFLAGS: -Ihwloc -I.
#include "hwloc.h"
#include "autogen/config.h"
#include <stdlib.h>

char* _hwloc_get_info_name_by_idx(hwloc_obj_t obj, int index) {
	if (index < obj->infos_count) {
		struct hwloc_info_s* info = &obj->infos[index];
		if (info)
		{
			return info->name;
		}
	}
	return NULL;
}
char* _hwloc_get_info_value_by_idx(hwloc_obj_t obj, int index) {
	if (index < obj->infos_count) {
		struct hwloc_info_s* info = &obj->infos[index];
		if (info)
		{
			return info->value;
		}
	}
	return NULL;
}

unsigned long _hwloc_read_numanode_memory(hwloc_obj_t obj) {
	if (obj->type == HWLOC_OBJ_NUMANODE && obj->attr != NULL)
	{
		return obj->attr->numanode.local_memory;
	}
	return 0;
}

int _hwloc_read_cache_data(hwloc_obj_t obj, unsigned long* size, unsigned int *depth, unsigned int *linesize, unsigned int *associativity, unsigned int *type) {
	if (obj->type >= HWLOC_OBJ_L1CACHE && obj->type <= HWLOC_OBJ_L5CACHE && obj->attr != NULL)
	{
		*size = obj->attr->cache.size;
		*depth = obj->attr->cache.depth;
		*linesize = obj->attr->cache.linesize;
		*associativity = obj->attr->cache.associativity;
		*type = obj->attr->cache.type;
		return 0;
	}
	return -1;
}

int _hwloc_read_pcidev_data(hwloc_obj_t obj, unsigned int* domain, unsigned int *bus, unsigned int *dev, unsigned int *func, unsigned int *class_id, \
	                        unsigned int *vendor_id, unsigned int *device_id, unsigned int *subvendor_id, unsigned int *subdevice_id, \
							unsigned int *revision) {
	if (obj->type >= HWLOC_OBJ_PCI_DEVICE && obj->attr != NULL)
	{
		*domain = (unsigned int)obj->attr->pcidev.domain;
		*bus = (unsigned int)obj->attr->pcidev.bus;
		*dev = (unsigned int)obj->attr->pcidev.dev;
		*func = (unsigned int)obj->attr->pcidev.func;
		*class_id = (unsigned int)obj->attr->pcidev.class_id;
		*vendor_id = (unsigned int)obj->attr->pcidev.vendor_id;
		*device_id = (unsigned int)obj->attr->pcidev.device_id;
		*subvendor_id = (unsigned int)obj->attr->pcidev.subvendor_id;
		*subdevice_id = (unsigned int)obj->attr->pcidev.subdevice_id;
		*revision = (unsigned int)obj->attr->pcidev.revision;
		return 0;
	}
	return -1;
}

int _hwloc_read_osdev_data(hwloc_obj_t obj, unsigned int* types)
{
	if (obj->type >= HWLOC_OBJ_OS_DEVICE && obj->attr != NULL)
	{
		*types = (unsigned int) obj->attr->osdev.type;
		return 0;
	}
	return -1;
}

hwloc_obj_t _hwloc_get_child(hwloc_obj_t obj, unsigned int offset) {
	if (offset < obj->arity)
	{
		return obj->children[offset];
	}
	return NULL;
}

hwloc_obj_t _hwloc_get_memory_child(hwloc_obj_t obj, unsigned int offset) {
	if (offset < obj->memory_arity)
	{
		return &obj->memory_first_child[offset];
	}
	return NULL;
}

hwloc_obj_t _hwloc_get_io_child(hwloc_obj_t obj, unsigned int offset) {
	if (offset < obj->io_arity)
	{
		return &obj->io_first_child[offset];
	}
	return NULL;
}

int _hwloc_cpuset_size(hwloc_cpuset_t cpuset) {
	return hwloc_bitmap_weight(cpuset);
}

int _hwloc_cpuset_isset(hwloc_cpuset_t cpuset, unsigned id) {
	return hwloc_bitmap_isset(cpuset, id);
}

*/
import "C"

const DEBUG bool = true

type HWLOC_OBJ_TYPE int

const (
	HWLOC_TYPE_MACHINE    HWLOC_OBJ_TYPE = C.HWLOC_OBJ_MACHINE
	HWLOC_TYPE_PACKAGE    HWLOC_OBJ_TYPE = C.HWLOC_OBJ_PACKAGE
	HWLOC_TYPE_CORE       HWLOC_OBJ_TYPE = C.HWLOC_OBJ_CORE
	HWLOC_TYPE_PU         HWLOC_OBJ_TYPE = C.HWLOC_OBJ_PU
	HWLOC_TYPE_L1CACHE    HWLOC_OBJ_TYPE = C.HWLOC_OBJ_L1CACHE
	HWLOC_TYPE_L2CACHE    HWLOC_OBJ_TYPE = C.HWLOC_OBJ_L2CACHE
	HWLOC_TYPE_L3CACHE    HWLOC_OBJ_TYPE = C.HWLOC_OBJ_L3CACHE
	HWLOC_TYPE_L4CACHE    HWLOC_OBJ_TYPE = C.HWLOC_OBJ_L4CACHE
	HWLOC_TYPE_L5CACHE    HWLOC_OBJ_TYPE = C.HWLOC_OBJ_L5CACHE
	HWLOC_TYPE_L1ICACHE   HWLOC_OBJ_TYPE = C.HWLOC_OBJ_L1ICACHE
	HWLOC_TYPE_L2ICACHE   HWLOC_OBJ_TYPE = C.HWLOC_OBJ_L2ICACHE
	HWLOC_TYPE_L3ICACHE   HWLOC_OBJ_TYPE = C.HWLOC_OBJ_L3ICACHE
	HWLOC_TYPE_GROUP      HWLOC_OBJ_TYPE = C.HWLOC_OBJ_GROUP
	HWLOC_TYPE_NUMANODE   HWLOC_OBJ_TYPE = C.HWLOC_OBJ_NUMANODE
	HWLOC_TYPE_PCI_DEVICE HWLOC_OBJ_TYPE = C.HWLOC_OBJ_PCI_DEVICE
	HWLOC_TYPE_OS_DEVICE  HWLOC_OBJ_TYPE = C.HWLOC_OBJ_OS_DEVICE
	HWLOC_TYPE_MEMCACHE   HWLOC_OBJ_TYPE = C.HWLOC_OBJ_MEMCACHE
	HWLOC_TYPE_MISC       HWLOC_OBJ_TYPE = C.HWLOC_OBJ_MISC
	HWLOC_TYPE_DIE        HWLOC_OBJ_TYPE = C.HWLOC_OBJ_DIE
	HWLOC_TYPE_MAX        HWLOC_OBJ_TYPE = C.HWLOC_OBJ_TYPE_MAX
)

type HWLOC_OBJ_OSDEV_TYPE int

const (
	HWLOC_OBJ_OSDEV_BLOCK       HWLOC_OBJ_OSDEV_TYPE = C.HWLOC_OBJ_OSDEV_BLOCK
	HWLOC_OBJ_OSDEV_GPU         HWLOC_OBJ_OSDEV_TYPE = C.HWLOC_OBJ_OSDEV_GPU
	HWLOC_OBJ_OSDEV_NETWORK     HWLOC_OBJ_OSDEV_TYPE = C.HWLOC_OBJ_OSDEV_NETWORK
	HWLOC_OBJ_OSDEV_OPENFABRICS HWLOC_OBJ_OSDEV_TYPE = C.HWLOC_OBJ_OSDEV_OPENFABRICS
	HWLOC_OBJ_OSDEV_DMA         HWLOC_OBJ_OSDEV_TYPE = C.HWLOC_OBJ_OSDEV_DMA
	HWLOC_OBJ_OSDEV_COPROC      HWLOC_OBJ_OSDEV_TYPE = C.HWLOC_OBJ_OSDEV_COPROC
)

type Object struct {
	Type           HWLOC_OBJ_TYPE `json:"type"`
	TypeString     string         `json:"typestring"`
	ID             uint           `json:"id"`
	IDString       string         `json:"idstring"`
	Depth          int            `json:"depth"`
	LogicalIndex   uint           `json:"logical_index"`
	parent         int64
	Infos          map[string]string `json:"infos"`
	Children       []Object          `json:"children"`
	MemoryChildren []Object          `json:"memory_children"`
	IOChildren     []Object          `json:"io_children"`
	HwlocObject    C.hwloc_obj_t
}

type CCTopologyFlags uint

const CCTOPOLOGY_NO_OSDEVICES CCTopologyFlags = (1 << 0)

type topology struct {
	root    Object
	objects []*Object
	flags   uint
}

type Topology interface {
	GetHwthreads() []uint
	GetHwthreadStrings() []string
	GetSockets() []uint
	GetSocketStrings() []string
	GetDies() []uint
	GetDieStrings() []string
	GetCores() []uint
	GetCoreStrings() []string
	GetMemoryDomains() []uint
	GetMemoryDomainStrings() []string
	GetPciDevices() []uint
	GetPciDeviceStrings() []string
	GetHwthreadsOfSocket(socket uint) []uint
	GetHwthreadStringsOfSocket(socket uint) []string
	GetHwthreadsOfMemoryDomain(memoryDomain uint) []uint
	GetHwthreadStringsOfMemoryDomain(memoryDomain uint) []string
	GetNumaNodeOfPciDevice(address string) int
	CpuInfo() CpuInformation
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(in []byte) error
}

var filterPciClasses = []string{
	"0x300",
	"0x880",
}

func skipPciDevice(obj Object) bool {
	if obj.Type != HWLOC_TYPE_PCI_DEVICE {
		return false
	}
	return !slices.Contains(filterPciClasses, obj.Infos["class_id"])
}

// func (t *HWLOC_OBJ_TYPE) String() string {
// 	return C.GoString(C.hwloc_obj_type_string(C.hwloc_obj_type_t(*t)))
// }

func (t *HWLOC_OBJ_TYPE) String() string {
	switch *t {
	case HWLOC_TYPE_MACHINE:
		return "node"
	case HWLOC_TYPE_PACKAGE:
		return "socket"
	case HWLOC_TYPE_CORE:
		return "core"
	case HWLOC_TYPE_PU:
		return "hwthread"
	case HWLOC_TYPE_NUMANODE:
		return "memoryDomain"
	case HWLOC_TYPE_DIE:
		return "die"
	case HWLOC_TYPE_PCI_DEVICE:
		return "accelerator"
	case HWLOC_TYPE_GROUP:
		return "group"
	case HWLOC_TYPE_MISC:
		return "misc"
	case HWLOC_TYPE_MEMCACHE:
		return "memcache"
	case HWLOC_TYPE_L1CACHE:
		return "l1cache"
	case HWLOC_TYPE_L2CACHE:
		return "l2cache"
	case HWLOC_TYPE_L3CACHE:
		return "l3cache"
	case HWLOC_TYPE_L4CACHE:
		return "l4cache"
	case HWLOC_TYPE_L5CACHE:
		return "l5cache"
	case HWLOC_TYPE_OS_DEVICE:
		return "osdevice"
	case HWLOC_TYPE_L1ICACHE:
		return "l1icache"
	case HWLOC_TYPE_L2ICACHE:
		return "l2icache"
	case HWLOC_TYPE_L3ICACHE:
		return "l3icache"
	}
	return "invalid"
}

func (t *HWLOC_OBJ_OSDEV_TYPE) String() string {
	switch *t {
	case HWLOC_OBJ_OSDEV_BLOCK:
		return "block"
	case HWLOC_OBJ_OSDEV_GPU:
		return "gpu"
	case HWLOC_OBJ_OSDEV_NETWORK:
		return "network"
	case HWLOC_OBJ_OSDEV_OPENFABRICS:
		return "openfabrics"
	case HWLOC_OBJ_OSDEV_DMA:
		return "dma"
	case HWLOC_OBJ_OSDEV_COPROC:
		return "accelerator"
	}
	return "invalid"
}

func (o Object) String() string {
	x, err := json.MarshalIndent(o, "", "  ")
	if err == nil {
		return string(x)
	}
	return "{}"
}

// func (t Topology) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(t.root)
// }

// func (t Topology) UnmarshalJSON(in []byte) error {
// 	return json.Unmarshal(in, &t.root)
// }

func (t *topology) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.root)
}

func addObject(o Object, objects *[]*Object) {
	*objects = append(*objects, &o)
	for _, c := range o.Children {
		addObject(c, objects)
	}
	for _, c := range o.MemoryChildren {
		addObject(c, objects)
	}
	for _, c := range o.IOChildren {
		addObject(c, objects)
	}
}

func (t *topology) UnmarshalJSON(in []byte) error {
	err := json.Unmarshal(in, &t.root)
	if err != nil {
		return err
	}
	addObject(t.root, &t.objects)
	return nil
}

func convertObject(hwloc_obj C.hwloc_obj_t) (Object, bool, error) {
	// ty := HWLOC_OBJ_TYPE(hwloc_obj._type)
	// fmt.Printf("HwlocObject Type %s with ID %d SubType '%s' Name '%s' LogIdx %d Infos %d\n", ty.String(), int(hwloc_obj.os_index), C.GoString(hwloc_obj.subtype), C.GoString(hwloc_obj.name), int(hwloc_obj.logical_index), int(hwloc_obj.infos_count))
	if HWLOC_OBJ_TYPE(hwloc_obj._type) >= HWLOC_TYPE_MAX {
		return Object{}, true, errors.New("invalid hwloc obj type")
	}
	o := Object{
		Type:           HWLOC_OBJ_TYPE(hwloc_obj._type),
		ID:             uint(hwloc_obj.os_index),
		LogicalIndex:   uint(hwloc_obj.logical_index),
		Depth:          int(hwloc_obj.depth),
		IDString:       fmt.Sprintf("%d", hwloc_obj.os_index),
		Infos:          make(map[string]string),
		Children:       make([]Object, 0),
		MemoryChildren: make([]Object, 0),
		IOChildren:     make([]Object, 0),
		HwlocObject:    hwloc_obj,
		parent:         -1,
	}
	o.TypeString = o.Type.String()
	if hwloc_obj.parent != nil {
		o.parent = int64(hwloc_obj.parent.gp_index)
	}
	for i := range int(hwloc_obj.infos_count) {
		name := C._hwloc_get_info_name_by_idx(hwloc_obj, C.int(i))
		value := C._hwloc_get_info_value_by_idx(hwloc_obj, C.int(i))
		o.Infos[C.GoString(name)] = C.GoString(value)
	}
	if (hwloc_obj.attr) != nil {
		switch hwloc_obj._type {
		case C.HWLOC_OBJ_NUMANODE:
			o.Infos["local_memory"] = fmt.Sprintf("%d", uint64(C._hwloc_read_numanode_memory(hwloc_obj)))
		case C.HWLOC_OBJ_L1CACHE, C.HWLOC_OBJ_L2CACHE, C.HWLOC_OBJ_L3CACHE, C.HWLOC_OBJ_L4CACHE, C.HWLOC_OBJ_L5CACHE:
			size := C.ulong(0)
			depth := C.uint(0)
			linesize := C.uint(0)
			associativity := C.uint(0)
			cachetype := C.uint(0)
			ret := C._hwloc_read_cache_data(hwloc_obj, &size, &depth, &linesize, &associativity, &cachetype)
			if ret == 0 {
				o.Infos["size"] = fmt.Sprintf("%d", size)
				o.Infos["depth"] = fmt.Sprintf("%d", depth)
				o.Infos["linesize"] = fmt.Sprintf("%d", linesize)
				o.Infos["associativity"] = fmt.Sprintf("%d", associativity)
				switch cachetype {
				case C.HWLOC_OBJ_CACHE_UNIFIED:
					o.Infos["type"] = "unified"
				case C.HWLOC_OBJ_CACHE_DATA:
					o.Infos["type"] = "data"
				case C.HWLOC_OBJ_CACHE_INSTRUCTION:
					o.Infos["type"] = "instruction"
				}
			}
		case C.HWLOC_OBJ_PCI_DEVICE:
			domain := C.uint(0)
			bus := C.uint(0)
			dev := C.uint(0)
			function := C.uint(0)
			class_id := C.uint(0)
			vendor_id := C.uint(0)
			device_id := C.uint(0)
			subvendor_id := C.uint(0)
			subdevice_id := C.uint(0)
			revision := C.uint(0)

			ret := C._hwloc_read_pcidev_data(hwloc_obj, &domain, &bus, &dev, &function, &class_id, &vendor_id, &device_id, &subvendor_id, &subdevice_id, &revision)
			if ret == 0 {
				o.Infos["domain"] = fmt.Sprintf("0x%04x", domain)
				o.Infos["bus"] = fmt.Sprintf("0x%02x", bus)
				o.Infos["dev"] = fmt.Sprintf("0x%02x", dev)
				o.Infos["func"] = fmt.Sprintf("0x%01x", function)
				o.Infos["class_id"] = fmt.Sprintf("0x%X", class_id)
				o.Infos["vendor_id"] = fmt.Sprintf("0x%X", vendor_id)
				o.Infos["device_id"] = fmt.Sprintf("0x%X", device_id)
				o.Infos["subvendor_id"] = fmt.Sprintf("0x%X", subvendor_id)
				o.Infos["subdevice_id"] = fmt.Sprintf("0x%X", subdevice_id)
				o.Infos["revision"] = fmt.Sprintf("%d", revision)
				o.Infos["pci_address"] = fmt.Sprintf("%04x:%02x:%02x.%01x", domain, bus, dev, function)
			}
		case C.HWLOC_OBJ_OS_DEVICE:
			types := C.uint(0)
			ret := C._hwloc_read_osdev_data(hwloc_obj, &types)
			if ret == 0 {
				mytypes := HWLOC_OBJ_OSDEV_TYPE(types)
				o.Infos["type"] = mytypes.String()
			}
		}
	}
	return o, false, nil
}

func (t *topology) additionalMachineOps(hwtopo C.hwloc_topology_t) []Object {
	out := make([]Object, 0)
	nbobj := C.hwloc_get_nbobjs_by_depth(hwtopo, C.HWLOC_TYPE_DEPTH_PCI_DEVICE)
	for j := range int(nbobj) {
		hwobj := C.hwloc_get_obj_by_depth(hwtopo, C.HWLOC_TYPE_DEPTH_PCI_DEVICE, C.uint(j))
		obj, skip, err := convertObject(hwobj)
		if skip || skipPciDevice(obj) {
			continue
		}
		if err != nil {
			continue
		}
		numa_file := filepath.Join("/sys/bus/pci/devices", obj.Infos["pci_address"], "numa_node")
		content, err := os.ReadFile(numa_file)
		if err == nil {
			numa_node, err := strconv.ParseInt(strings.TrimSpace(string(content)), 10, 64)
			if err == nil {
				if numa_node < 0 {
					out = append(out, obj)
				}
			}
		}
	}
	nbobj = C.hwloc_get_nbobjs_by_depth(hwtopo, C.HWLOC_TYPE_DEPTH_OS_DEVICE)
	for j := range int(nbobj) {
		hwobj := C.hwloc_get_obj_by_depth(hwtopo, C.HWLOC_TYPE_DEPTH_OS_DEVICE, C.uint(j))
		obj, skip, err := convertObject(hwobj)
		if err == nil && !skip {
			out = append(out, obj)
		}
	}
	return out
}

func (t *topology) additionalPackageOps(hwtopo C.hwloc_topology_t, packageObj Object) []Object {
	out := make([]Object, 0)
	nbobj := C.hwloc_get_nbobjs_by_depth(hwtopo, C.HWLOC_TYPE_DEPTH_NUMANODE)
	for j := range int(nbobj) {
		hwobj := C.hwloc_get_obj_by_depth(hwtopo, C.HWLOC_TYPE_DEPTH_NUMANODE, C.uint(j))
		if HWLOC_OBJ_TYPE(hwobj.parent._type) == packageObj.Type &&
			int64(hwobj.parent.os_index) == int64(packageObj.ID) &&
			int(hwobj.parent.logical_index) == int(packageObj.LogicalIndex) {
			out = append(out, t.traverseObject(hwtopo, hwobj))
		}
	}
	nbobj = C.hwloc_get_nbobjs_by_depth(hwtopo, C.HWLOC_TYPE_DEPTH_OS_DEVICE)
	for j := range int(nbobj) {
		hwobj := C.hwloc_get_obj_by_depth(hwtopo, C.HWLOC_TYPE_DEPTH_OS_DEVICE, C.uint(j))
		if HWLOC_OBJ_TYPE(hwobj.parent._type) == packageObj.Type &&
			int64(hwobj.parent.os_index) == int64(packageObj.ID) &&
			int(hwobj.parent.logical_index) == int(packageObj.LogicalIndex) {
			out = append(out, t.traverseObject(hwtopo, hwobj))
		}
	}
	return out
}

func (t *topology) additionalNumaOps(hwtopo C.hwloc_topology_t, numaObj Object) []Object {
	out := make([]Object, 0)
	nbobj := C.hwloc_get_nbobjs_by_depth(hwtopo, C.HWLOC_TYPE_DEPTH_PCI_DEVICE)
	for j := range int(nbobj) {
		hwobj := C.hwloc_get_obj_by_depth(hwtopo, C.HWLOC_TYPE_DEPTH_PCI_DEVICE, C.uint(j))
		obj, skip, err := convertObject(hwobj)
		if skip || skipPciDevice(obj) {
			continue
		}
		if err != nil {
			continue
		}
		numa_file := filepath.Join("/sys/bus/pci/devices", obj.Infos["pci_address"], "numa_node")
		content, err := os.ReadFile(numa_file)
		if err == nil {
			numa_node, err := strconv.ParseInt(strings.TrimSpace(string(content)), 10, 64)
			if err == nil {
				if numa_node == int64(numaObj.ID) {
					out = append(out, obj)
				}
			}
		}
	}
	nbobj = C.hwloc_get_nbobjs_by_depth(hwtopo, C.HWLOC_TYPE_DEPTH_OS_DEVICE)
	for j := range int(nbobj) {
		hwobj := C.hwloc_get_obj_by_depth(hwtopo, C.HWLOC_TYPE_DEPTH_OS_DEVICE, C.uint(j))
		if HWLOC_OBJ_TYPE(hwobj.parent._type) == numaObj.Type &&
			int64(hwobj.parent.os_index) == int64(numaObj.ID) &&
			int(hwobj.parent.logical_index) == int(numaObj.LogicalIndex) {
			out = append(out, t.traverseObject(hwtopo, hwobj))
		}
	}
	return out
}

func (t *topology) traverseObject(hwtopo C.hwloc_topology_t, hwloc_obj C.hwloc_obj_t) Object {
	obj, skip, err := convertObject(hwloc_obj)
	if skip || skipPciDevice(obj) || err != nil {
		return Object{Type: HWLOC_TYPE_MAX}
	}

	for i := range int(hwloc_obj.arity) {
		obj.Children = append(obj.Children, t.traverseObject(hwtopo, C._hwloc_get_child(hwloc_obj, C.uint(i))))
	}
	for i := range int(hwloc_obj.memory_arity) {
		obj.MemoryChildren = append(obj.MemoryChildren, t.traverseObject(hwtopo, C._hwloc_get_memory_child(hwloc_obj, C.uint(i))))
	}
	// for i := range int(hwloc_obj.io_arity) {
	// 	obj.IOChildren = append(obj.IOChildren, t.traverseObject(hwtopo, C._hwloc_get_io_child(hwloc_obj, C.uint(i))))
	// }
	switch obj.Type {
	case HWLOC_TYPE_MACHINE:
		obj.Children = append(obj.Children, t.additionalMachineOps(hwtopo)...)
	case HWLOC_TYPE_PACKAGE:
		obj.Children = append(obj.Children, t.additionalPackageOps(hwtopo, obj)...)
	case HWLOC_TYPE_NUMANODE:
		obj.Children = append(obj.Children, t.additionalNumaOps(hwtopo, obj)...)
	}
	t.objects = append(t.objects, &obj)
	return obj
}

func (t *topology) init() error {
	var hwtopo C.hwloc_topology_t
	ret := C.hwloc_topology_init(&hwtopo)
	if ret != 0 {
		return fmt.Errorf("hwloc_topology_init returned %d", ret)
	}
	C.hwloc_topology_set_flags(hwtopo, C.HWLOC_TOPOLOGY_FLAG_INCLUDE_DISALLOWED)
	C.hwloc_topology_set_all_types_filter(hwtopo, C.HWLOC_TYPE_FILTER_KEEP_ALL)
	ret = C.hwloc_topology_load(hwtopo)
	if ret != 0 {
		return fmt.Errorf("hwloc_topology_load returned %d", ret)
	}

	t.objects = make([]*Object, 0)

	rootobj := C.hwloc_get_root_obj(hwtopo)

	newroot := t.traverseObject(hwtopo, rootobj)

	t.root = newroot

	return nil
}

var _ccTopology_local_topo *topology = nil

func LocalTopology() (Topology, error) {
	if _ccTopology_local_topo == nil {
		t := new(topology)
		err := t.init()
		if err == nil {
			_ccTopology_local_topo = t
		}
		return t, err
	} else {
		return _ccTopology_local_topo, nil
	}
}

func RemoteTopology(topologyJson json.RawMessage) (Topology, error) {
	var t topology
	t.objects = make([]*Object, 0)
	err := t.UnmarshalJSON(topologyJson)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal topology JSON: %v", err.Error())
	}
	return &t, nil
}

func (t *topology) getTypeObjectIds(Type HWLOC_OBJ_TYPE) []uint {
	out := make([]uint, 0)
	for _, o := range t.objects {
		if o.Type == Type {
			out = append(out, o.ID)
		}
	}
	return out
}

func (t *topology) getTypeObjectIdStrings(Type HWLOC_OBJ_TYPE) []string {
	out := make([]string, 0)
	for _, o := range t.objects {
		if o.Type == Type {
			out = append(out, o.IDString)
		}
	}
	return out
}

func getChildrenIdsOfType(o Object, t HWLOC_OBJ_TYPE, out *[]uint) {
	if o.Type == t {
		*out = append(*out, o.ID)
	}
	for _, c := range o.Children {
		getChildrenIdsOfType(c, t, out)
	}
}

func getChildrenIdStringsOfType(o Object, t HWLOC_OBJ_TYPE, out *[]string) {
	if o.Type == t {
		*out = append(*out, o.IDString)
	}
	for _, c := range o.Children {
		getChildrenIdStringsOfType(c, t, out)
	}
}

func (t *topology) GetHwthreads() []uint {
	return t.getTypeObjectIds(HWLOC_TYPE_PU)
}

func (t *topology) GetHwthreadStrings() []string {
	return t.getTypeObjectIdStrings(HWLOC_TYPE_PU)
}

func (t *topology) GetCores() []uint {
	return t.getTypeObjectIds(HWLOC_TYPE_CORE)
}

func (t *topology) GetCoreStrings() []string {
	return t.getTypeObjectIdStrings(HWLOC_TYPE_CORE)
}

func (t *topology) GetSockets() []uint {
	return t.getTypeObjectIds(HWLOC_TYPE_PACKAGE)
}

func (t *topology) GetSocketStrings() []string {
	return t.getTypeObjectIdStrings(HWLOC_TYPE_PACKAGE)
}

func (t *topology) GetDies() []uint {
	out := t.getTypeObjectIds(HWLOC_TYPE_DIE)
	if len(out) == 0 {
		out = t.getTypeObjectIds(HWLOC_TYPE_PACKAGE)
	}
	return out
}

func (t *topology) GetDieStrings() []string {
	out := t.getTypeObjectIdStrings(HWLOC_TYPE_DIE)
	if len(out) == 0 {
		out = t.getTypeObjectIdStrings(HWLOC_TYPE_PACKAGE)
	}
	return out
}

func (t *topology) GetMemoryDomains() []uint {
	return t.getTypeObjectIds(HWLOC_TYPE_NUMANODE)
}

func (t *topology) GetMemoryDomainStrings() []string {
	return t.getTypeObjectIdStrings(HWLOC_TYPE_NUMANODE)
}

func (t *topology) GetPciDevices() []uint {
	out := make([]uint, 0)
	for _, o := range t.objects {
		if o.Type == HWLOC_TYPE_MACHINE || o.Type == HWLOC_TYPE_PACKAGE || o.Type == HWLOC_TYPE_DIE || o.Type == HWLOC_TYPE_NUMANODE {
			for _, c := range o.Children {
				if c.Type == HWLOC_TYPE_PCI_DEVICE {
					out = append(out, c.ID)
				}
			}
		}
	}
	return out
}

func (t *topology) GetPciDeviceStrings() []string {
	out := make([]string, 0)
	for _, o := range t.objects {
		if o.Type == HWLOC_TYPE_MACHINE || o.Type == HWLOC_TYPE_PACKAGE || o.Type == HWLOC_TYPE_DIE || o.Type == HWLOC_TYPE_NUMANODE {
			for _, c := range o.Children {
				if c.Type == HWLOC_TYPE_PCI_DEVICE {
					if addr, ok := c.Infos["pci_address"]; ok {
						out = append(out, addr)
					}
				}
			}
		}
	}
	return out
}

func (t *topology) GetHwthreadsOfSocket(socket uint) []uint {
	out := make([]uint, 0)
	for _, o := range t.objects {
		if o.Type == HWLOC_TYPE_PACKAGE && o.ID == socket {
			getChildrenIdsOfType(*o, HWLOC_TYPE_PU, &out)
		}
	}
	return out
}

func (t *topology) GetHwthreadStringsOfSocket(socket uint) []string {
	out := make([]string, 0)
	for _, o := range t.objects {
		if o.Type == HWLOC_TYPE_PACKAGE && o.ID == socket {
			getChildrenIdStringsOfType(*o, HWLOC_TYPE_PU, &out)
		}
	}
	return out
}

func (t *topology) GetHwthreadsOfMemoryDomain(memoryDomain uint) []uint {
	out := make([]uint, 0)
	for _, o := range t.objects {
		if o.Type == HWLOC_TYPE_NUMANODE {
			numHWthreads := C._hwloc_cpuset_size(o.HwlocObject.cpuset)
			for _, hwo := range t.objects {
				if hwo.Type == HWLOC_TYPE_PU && C._hwloc_cpuset_isset(o.HwlocObject.cpuset, C.unsigned(hwo.ID)) == 1 {
					out = append(out, hwo.ID)
				}
				if len(out) == int(numHWthreads) {
					break
				}
			}
		}
	}
	return out
}

func (t *topology) GetHwthreadStringsOfMemoryDomain(memoryDomain uint) []string {
	out := make([]string, 0)
	for _, o := range t.objects {
		if o.Type == HWLOC_TYPE_NUMANODE {
			numHWthreads := C._hwloc_cpuset_size(o.HwlocObject.cpuset)
			for _, hwo := range t.objects {
				if hwo.Type == HWLOC_TYPE_PU && C._hwloc_cpuset_isset(o.HwlocObject.cpuset, C.unsigned(hwo.ID)) == 1 {
					out = append(out, fmt.Sprintf("%d", hwo.ID))
				}
				if len(out) == int(numHWthreads) {
					break
				}
			}
		}
	}
	return out
}

func (t *topology) GetNumaNodeOfPciDevice(address string) int {
	for _, o := range t.objects {
		if o.Type == HWLOC_TYPE_NUMANODE {
			for _, c := range o.Children {
				if c.Type == HWLOC_TYPE_PCI_DEVICE {
					if addr, ok := c.Infos["pci_address"]; ok && addr == address {
						return int(o.ID)
					}
				}
			}
		}
	}
	return -1
}

type CpuInformation struct {
	NumHWthreads   int
	SMTWidth       int
	NumSockets     int
	NumDies        int
	NumCores       int
	NumNumaDomains int
}

func (t *topology) CpuInfo() CpuInformation {
	return CpuInformation{
		NumHWthreads:   len(t.GetHwthreads()),
		SMTWidth:       len(t.GetHwthreads()) / len(t.GetCores()),
		NumSockets:     len(t.GetSockets()),
		NumDies:        len(t.GetDies()),
		NumCores:       len(t.GetCores()),
		NumNumaDomains: len(t.GetMemoryDomains()),
	}
}
