//  Copyright (C) 2026 NodeByte LTD

package state

import "strings"

func IsAdmin(userId string) bool {
	if userId == "" || Config.AdminUserIDs == "" {
		return false
	}

	for _, id := range strings.Split(Config.AdminUserIDs, ",") {
		if strings.TrimSpace(id) == userId {
			return true
		}
	}

	return false
}
