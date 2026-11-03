// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

#define _GNU_SOURCE
#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include "hwloc.h"
static unsigned initialized, failed, loaded, destroyed;
static void trace(void) {
 const char *path=getenv("CC_TOPOLOGY_LIFETIME_TRACE");
 if (!path) return;
 FILE *file=fopen(path,"w");if (!file) abort();
 fprintf(file,"{\"initialized\":%u,\"failed\":%u,\"loaded\":%u,\"destroyed\":%u}",initialized,failed,loaded,destroyed);
 fclose(file);
}
int hwloc_topology_init(hwloc_topology_t *topology) {
 int (*next)(hwloc_topology_t *)=dlsym(RTLD_NEXT,"hwloc_topology_init");
 int result=next(topology);if(result==0) initialized++;trace();return result;
}
int hwloc_topology_load(hwloc_topology_t topology) {
 int (*next)(hwloc_topology_t)=dlsym(RTLD_NEXT,"hwloc_topology_load");
 int result=next(topology);if(result==0) loaded++;else failed++;trace();return result;
}
void hwloc_topology_destroy(hwloc_topology_t topology) {
 void (*next)(hwloc_topology_t)=dlsym(RTLD_NEXT,"hwloc_topology_destroy");
 next(topology);destroyed++;trace();
}
