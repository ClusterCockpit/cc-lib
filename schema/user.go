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
	RoleAPI                   // API access (programmatic/service accounts)
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

// HasProject reports whether the user is authorized for the given project name.
func (u *User) HasProject(project string) bool {
	return slices.Contains(u.Projects, project)
}

// GetRoleString returns the lowercase string representation of a Role enum value.
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
		return RoleAPI
	case "anonymous":
		return RoleAnonymous
	default:
		return RoleError
	}
}

// IsValidRole reports whether the given string corresponds to a known role name.
func IsValidRole(role string) bool {
	return getRoleEnum(role) != RoleError
}

// HasValidRole checks whether the user has the specified role and whether the role string is valid.
// Returns hasRole=true if the user has the role, and isValid=true if the role name is recognized.
func (u *User) HasValidRole(role string) (hasRole bool, isValid bool) {
	if IsValidRole(role) {
		if slices.Contains(u.Roles, role) {
			return true, true
		}
		return false, true
	}
	return false, false
}

// HasRole reports whether the user has the specified role.
func (u *User) HasRole(role Role) bool {
	return slices.Contains(u.Roles, GetRoleString(role))
}

// HasAnyRole reports whether the user has at least one of the given roles.
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

// HasAllRoles reports whether the user has every one of the given roles.
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

// HasNotRoles reports whether the user has none of the given roles.
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

// GetValidRoles returns the list of assignable role names. Only admins may call this;
// returns an error if the user does not have the Admin role.
func GetValidRoles(user *User) ([]string, error) {
	var vals []string
	if user.HasRole(RoleAdmin) {
		for i := RoleAPI; i < RoleError; i++ {
			vals = append(vals, GetRoleString(i))
		}
		return vals, nil
	}

	return vals, fmt.Errorf("%s: only admins are allowed to fetch a list of roles", user.Username)
}

// GetValidRolesMap returns a map of role names to Role enum values. Requires any
// authenticated (non-anonymous) user; returns an error for anonymous users.
func GetValidRolesMap(user *User) (map[string]Role, error) {
	named := make(map[string]Role)
	if user.HasNotRoles([]Role{RoleAnonymous}) {
		for i := RoleAPI; i < RoleError; i++ {
			named[GetRoleString(i)] = i
		}
		return named, nil
	}
	return named, fmt.Errorf("only known users are allowed to fetch a list of roles")
}

// GetAuthLevel returns the user's highest-privilege role.
// Returns RoleError if the user has no recognized roles.
func (u *User) GetAuthLevel() Role {
	if u.HasRole(RoleAdmin) {
		return RoleAdmin
	} else if u.HasRole(RoleSupport) {
		return RoleSupport
	} else if u.HasRole(RoleManager) {
		return RoleManager
	} else if u.HasRole(RoleUser) {
		return RoleUser
	} else if u.HasRole(RoleAPI) {
		return RoleAPI
	} else if u.HasRole(RoleAnonymous) {
		return RoleAnonymous
	} else {
		return RoleError
	}
}
