package middleware

import (
	"reflect"
	"testing"

	"github.com/GyroZepelix/mithril-cms/internal/constant"
)

func TestRegisterRole(t *testing.T) {
	authorPermissions := []AccessPermission{
		{
			ResourceTypePost,
			CanCreate,
		},
		{
			ResourceTypePost,
			CanDelete,
		},
		{
			ResourceTypePost,
			CanUpdate,
		},
	}

	adminPermissions := append(authorPermissions,
		[]AccessPermission{
			{
				ResourceTypePost,
				CanDeleteAll,
			},
		}...,
	)

	pm := NewPermissionManager()
	pm.RegisterRole(constant.UserRoleAuthor, authorPermissions...)
	pm.RegisterRole(constant.UserRoleAdmin, adminPermissions...)

	if !reflect.DeepEqual(pm.registeredPermissions[constant.UserRoleAuthor], authorPermissions) {
		t.Error("expected author roles to match the passed in roles", pm.registeredPermissions[constant.UserRoleAuthor], authorPermissions)
	}
	if !reflect.DeepEqual(pm.registeredPermissions[constant.UserRoleAdmin], adminPermissions) {
		t.Error("expected author roles to match the passed in roles", pm.registeredPermissions[constant.UserRoleAdmin], adminPermissions)
	}
}

func TestValidatePermission(t *testing.T) {

	authorPermissions := []AccessPermission{
		{
			ResourceTypePost,
			CanCreate,
		},
		{
			ResourceTypePost,
			CanDelete,
		},
		{
			ResourceTypePost,
			CanUpdate,
		},
	}

	adminPermissions := append(authorPermissions,
		[]AccessPermission{
			{
				ResourceTypePost,
				CanDeleteAll,
			},
		}...,
	)

	pm := NewPermissionManager()
	pm.RegisterRole(constant.UserRoleAuthor, authorPermissions...)
	pm.RegisterRole(constant.UserRoleAdmin, adminPermissions...)

	t.Run("Should allow author to create post", func(t *testing.T) {
		givenRole := constant.UserRoleAuthor
		requiredPermissions := AccessPermission{
			ResourceTypePost,
			CanCreate,
		}

		if pm.ValidatePermission(givenRole, requiredPermissions) == false {
			t.Error("Author should have been permitted but was not")
		}
	})

	t.Run("Should dissalow author to delete other users post", func(t *testing.T) {
		givenRole := constant.UserRoleAuthor
		requiredPermissions := AccessPermission{
			ResourceTypePost,
			CanDeleteAll,
		}

		if pm.ValidatePermission(givenRole, requiredPermissions) == true {
			t.Error("Author should have been permitted but was not")
		}
	})

	t.Run("Should allow admin to delete other users post", func(t *testing.T) {
		givenRole := constant.UserRoleAdmin
		requiredPermissions := AccessPermission{
			ResourceTypePost,
			CanDeleteAll,
		}

		if pm.ValidatePermission(givenRole, requiredPermissions) == false {
			t.Error("Admin should have been permitted but was not")
		}
	})
}
