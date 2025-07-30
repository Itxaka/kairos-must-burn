//go:build windows

package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os/user"
)

func CheckElevatedPermissions() (bool, error) {
	//https://github.com/golang/go/issues/28804#issuecomment-505326268
	var sid *windows.SID
	var userIsAdmin bool
	// https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-checktokenmembership

	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)

	if err != nil {
		return false, fmt.Errorf("Error while checking for elevated permissions: %s", err)
	}

	//We must free the sid to prevent security token leaks
	defer windows.FreeSid(sid)
	token := windows.Token(0)
	userIsAdmin, err = isAdmin()

	if err != nil {
		return false, fmt.Errorf("Error while checking admin group membership: %s", err)
	}

	// Run as administrator
	member, err := token.IsMember(sid)

	if err != nil {
		return false, fmt.Errorf("Error while checking for elevated permissions: %s", err)
	}

	if !member {
		return false, fmt.Errorf("we need to run with administrator permissions. Run as admin is: %t, User in admin group: %t", member, userIsAdmin)

	}

	return true, nil
}

func isAdmin() (bool, error) {
	// Get current user
	u, err := user.Current()
	if err != nil {
		return false, fmt.Errorf("error retrieving current user: %s", err)
	}
	// Get IDs of group user is a member of
	ids, err := u.GroupIds()
	if err != nil {
		return false, fmt.Errorf("error retrieving group ids: %s", err)
	}

	// Check for existence of BUILTIN\ADMINISTRATORS group id
	for i := range ids {
		// BUILTIN\ADMINISTRATORS
		if "S-1-5-32-544" == ids[i] {
			return true, nil
		}

	}

	return false, nil
}
