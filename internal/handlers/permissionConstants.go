package handlers

import "github.com/GyroZepelix/mithril-cms/internal/service/permission"

var (
	readUserOwned = permission.AccessPermission{
		ResourceType:    permission.ResourceTypeUser,
		Permission:      permission.CanRead,
		PermissionLevel: permission.Owned,
	}
	readUserAll = permission.AccessPermission{
		ResourceType:    permission.ResourceTypeUser,
		Permission:      permission.CanRead,
		PermissionLevel: permission.All,
	}
)
