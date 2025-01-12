package permission

import (
	"context"
	"testing"

	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
)

func TestIsOwner(t *testing.T) {
	oc := NewOwnershipChecker(MockContentManager{})

	t.Run("Should pass when checking ownership of user", func(t *testing.T) {
		ctx := context.Background()
		userId := "1"
		userResourceId := "1"

		isOwner, err := oc.IsOwner(userId, ResourceTypeUser, userResourceId, ctx)
		if err != nil {
			t.Error("Failed calling IsOwner:", err)
		}
		if isOwner == false {
			t.Error("User should be owner of his profile")
		}

	})

	t.Run("Should fail when checking ownership of user", func(t *testing.T) {
		ctx := context.Background()
		userId := "1"
		userResourceId := "2"

		isOwner, err := oc.IsOwner(userId, ResourceTypeUser, userResourceId, ctx)
		if err != nil {
			t.Error("Failed calling IsOwner:", err)
		}
		if isOwner {
			t.Error("User should not be owner of another profile")
		}

	})

	t.Run("Should pass when checking ownership of posts", func(t *testing.T) {
		ctx := context.Background()
		userId := "1"
		postResourceId := "91"

		isOwner, err := oc.IsOwner(userId, ResourceTypePost, postResourceId, ctx)
		if err != nil {
			t.Error("Failed calling IsOwner:", err)
		}
		if isOwner == false {
			t.Error("User should be owner of his post")
		}

	})

	t.Run("Should fail when checking ownership of posts", func(t *testing.T) {
		ctx := context.Background()
		userId := "1"
		postResourceId := "92"

		isOwner, err := oc.IsOwner(userId, ResourceTypePost, postResourceId, ctx)
		if err != nil {
			t.Error("Failed calling IsOwner:", err)
		}
		if isOwner {
			t.Error("User should not be owner of another post")
		}

	})

	// t.Run("Should pass when checking ownership of comments", func(t *testing.T) {
	// 	userId := "1"
	// 	commentResourceId := "owner"
	//
	// 	isOwner, err := oc.IsOwner(userId, ResourceTypePost, commentResourceId, ctx)
	// 	if err != nil {
	// 		t.Error("Failed calling IsOwner:", err)
	// 	}
	// 	if isOwner == false {
	// 		t.Error("User should be owner of his post")
	// 	}
	//
	// })
	//
	// t.Run("Should fail when checking ownership of comments", func(t *testing.T) {
	// 	userId := "1"
	// 	commentResourceId := "notowner"
	//
	// 	isOwner, err := oc.IsOwner(userId, ResourceTypePost, commentResourceId, ctx)
	// 	if err != nil {
	// 		t.Error("Failed calling IsOwner:", err)
	// 	}
	// 	if isOwner {
	// 		t.Error("User should not be owner of another post")
	// 	}
	//
	// })
}

// -- Mocks --

type MockContentManager struct{}

// GetContent implements content.Manager.
func (m MockContentManager) GetContent(contentId int32, ctx context.Context) (persistence.Post, error) {
	if contentId == 91 {
		return persistence.Post{
			AuthorID: 1,
		}, nil
	} else {
		return persistence.Post{
			AuthorID: 2,
		}, nil
	}
}
