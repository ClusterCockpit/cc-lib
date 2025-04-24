<!--
---
title: Hostlist expansion
description: Package to expand hostlists like 'n[0-1],m[2-3]'
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/hostlist/_index.md
---
-->

# Hostlist

This package provides hostlist expansion for host specifications like `n[1-2],m[1,2]` or `n[1-2]-p,n[1,2]-q`.
The content of the `[]` can be a mixture of comma-separated entries and `X-Y` ranges.
Only a single specification of `[]` per entry is allowed.

While expanding to the hostlist duplicated hosts are removed, so `n[1-2],n[1,2]` (twice the same specification) results in `[n1, n2]`.
Zeros in the range specifications are preserved: `n[01-04,06-07,09]` -> `[n01, n02, n03, n04, n06, n07, n09]`

Invalid specifications:
- More than one `-` in a range: `[1-2-2]`
- Only increasing ranges, no `[2-1]`
- Other symbols than `-` and `,` in `[]` are not allowed, e.g. `@`
