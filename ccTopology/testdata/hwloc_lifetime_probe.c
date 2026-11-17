// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

#define _GNU_SOURCE
#include <dlfcn.h>
#include <errno.h>
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
 /* hwloc falls back to native discovery when an env-selected backend (e.g. HWLOC_XMLFILE)
    fails, so a load failure cannot be provoked reliably from outside. Inject it here. */
 const char *fail=getenv("CC_TOPOLOGY_LIFETIME_FAIL_LOAD");
 if (fail && fail[0]=='1') { failed++; trace(); errno=EINVAL; return -1; }
 int (*next)(hwloc_topology_t)=dlsym(RTLD_NEXT,"hwloc_topology_load");
 int result=next(topology);if(result==0) loaded++;else failed++;trace();return result;
}
void hwloc_topology_destroy(hwloc_topology_t topology) {
 void (*next)(hwloc_topology_t)=dlsym(RTLD_NEXT,"hwloc_topology_destroy");
 next(topology);destroyed++;trace();
}
