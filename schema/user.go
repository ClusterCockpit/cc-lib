// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package schema

import (
	"fmt"
	"slices"
	"strings"
)

// Role defines the authorization level for a user in ClusterCockpit.
// Roles form a hierarchy with increasing privileges: Anonymous < Api < User < Manager < Support < Admin.
type Role int

const (
	RoleAnonymous Role = iota // Unauthenticated or guest access
	RoleApi                   // API access (programmatic/service accounts)
	RoleUser                  // Regular user (can view own jobs)
	RoleManager               // Project manager (can view project jobs)
	RoleSupport               // Support staff (can view all jobs, limited admin)
	RoleAdmin                 // Full administrator access
	RoleError                 // Invalid/error role
)

// AuthSource identifies the authentication backend that validated a user.
type AuthSource int

const (
	AuthViaLocalPassword AuthSource = iota // Local database password authentication
	AuthViaLDAP                            // LDAP directory authentication
	AuthViaToken                           // JWT or API token authentication
	AuthViaOIDC                            // OpenID Connect authentication
	AuthViaAll                             // Accepts any auth source (special case)
)

// AuthType distinguishes between different authentication contexts.
type AuthType int

const (
	AuthToken   AuthType = iota // API token-based authentication
	AuthSession                 // Session cookie-based authentication
)

// User represents a ClusterCockpit user account with authentication and authorization information.
//
// Users are authenticated via various sources (local, LDAP, OIDC) and assigned roles that
// determine access levels. Projects lists the HPC projects/accounts the user has access to.
type User struct {
	Username   string     `json:"username"`   // Unique username
	Password   string     `json:"-"`          // Password hash (never serialized to JSON)
	Name       string     `json:"name"`       // Full display name
	Email      string     `json:"email"`      // Email address
	Roles      []string   `json:"roles"`      // Assigned role names
	Projects   []string   `json:"projects"`   // Authorized project/account names
	AuthType   AuthType   `json:"authType"`   // How the user authenticated
	AuthSource AuthSource `json:"authSource"` // Which system authenticated the user
}

func (u *User) HasProject(project string) bool {
	return slices.Contains(u.Projects, project)
}

func GetRoleString(roleInt Role) string {
	return [6]string{"anonymous", "api", "user", "manager", "support", "admin"}[roleInt]
}

func getRoleEnum(roleStr string) Role {
	switch strings.ToLower(roleStr) {
	case "admin":
		return RoleAdmin
	case "support":
		return RoleSupport
	case "manager":
		return RoleManager
	case "user":
		return RoleUser
	case "api":
		return RoleApi
	case "anonymous":
		return RoleAnonymous
	default:
		return RoleError
	}
}

func IsValidRole(role string) bool {
	return getRoleEnum(role) != RoleError
}

// Check if User has SPECIFIED role AND role is VALID
func (u *User) HasValidRole(role string) (hasRole bool, isValid bool) {
	if IsValidRole(role) {
		if slices.Contains(u.Roles, role) {
			return true, true
		}
		return false, true
	}
	return false, false
}

// Check if User has SPECIFIED role
func (u *User) HasRole(role Role) bool {
	return slices.Contains(u.Roles, GetRoleString(role))
}

// Check if User has ANY of the listed roles
func (u *User) HasAnyRole(queryroles []Role) bool {
	for _, ur := range u.Roles {
		for _, qr := range queryroles {
			if ur == GetRoleString(qr) {
				return true
			}
		}
	}
	return false
}

// Check if User has ALL of the listed roles
func (u *User) HasAllRoles(queryroles []Role) bool {
	target := len(queryroles)
	matches := 0
	for _, ur := range u.Roles {
		for _, qr := range queryroles {
			if ur == GetRoleString(qr) {
				matches += 1
				break
			}
		}
	}

	if matches == target {
		return true
	} else {
		return false
	}
}

// Check if User has NONE of the listed roles
func (u *User) HasNotRoles(queryroles []Role) bool {
	matches := 0
	for _, ur := range u.Roles {
		for _, qr := range queryroles {
			if ur == GetRoleString(qr) {
				matches += 1
				break
			}
		}
	}

	if matches == 0 {
		return true
	} else {
		return false
	}
}

// Called by API endpoint '/roles/' from frontend: Only required for admin config -> Check Admin Role
func GetValidRoles(user *User) ([]string, error) {
	var vals []string
	if user.HasRole(RoleAdmin) {
		for i := RoleApi; i < RoleError; i++ {
			vals = append(vals, GetRoleString(i))
		}
		return vals, nil
	}

	return vals, fmt.Errorf("%s: only admins are allowed to fetch a list of roles", user.Username)
}

// Called by routerConfig web.page setup in backend: Only requires known user
func GetValidRolesMap(user *User) (map[string]Role, error) {
	named := make(map[string]Role)
	if user.HasNotRoles([]Role{RoleAnonymous}) {
		for i := RoleApi; i < RoleError; i++ {
			named[GetRoleString(i)] = i
		}
		return named, nil
	}
	return named, fmt.Errorf("only known users are allowed to fetch a list of roles")
}

// Find highest role
func (u *User) GetAuthLevel() Role {
	if u.HasRole(RoleAdmin) {
		return RoleAdmin
	} else if u.HasRole(RoleSupport) {
		return RoleSupport
	} else if u.HasRole(RoleManager) {
		return RoleManager
	} else if u.HasRole(RoleUser) {
		return RoleUser
	} else if u.HasRole(RoleApi) {
		return RoleApi
	} else if u.HasRole(RoleAnonymous) {
		return RoleAnonymous
	} else {
		return RoleError
	}
}
