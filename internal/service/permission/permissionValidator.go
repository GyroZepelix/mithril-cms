package permission

import (
	"slices"

	"github.com/GyroZepelix/mithril-cms/internal/constant"
)

type PermissionValidator interface {
	ValidatePermission(role constant.UserRole, permission AccessPermission) bool
	RegisterRole(role constant.UserRole, permissions ...AccessPermission)
}

type PermissionService struct {
	registeredPermissions map[constant.UserRole][]AccessPermission
}

func NewPermissionValidator() *PermissionService {
	return &PermissionService{
		registeredPermissions: make(map[constant.UserRole][]AccessPermission),
	}
}

func (pm *PermissionService) RegisterRole(role constant.UserRole, permissions ...AccessPermission) {
	pm.registeredPermissions[role] = permissions
}

func (m *PermissionService) ValidatePermission(role constant.UserRole, permission AccessPermission) bool {
	userPermissions := m.registeredPermissions[role]
	if userPermissions == nil {
		return false
	}

	if slices.Contains(userPermissions, permission) {
		return true
	}
	return false
}
