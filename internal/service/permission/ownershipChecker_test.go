package permission

import (
	"context"
	"testing"

	"github.com/GyroZepelix/mithril-cms/internal/service/content/mocks"
	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func setupTest(t *testing.T) (*mock_content.MockManager, OwnershipChecker) {
	ctrl := gomock.NewController(t)
	mockManager := mock_content.NewMockManager(ctrl)
	oc := NewOwnershipChecker(mockManager)
	return mockManager, oc
}

func TestIsOwner(t *testing.T) {
	t.Run("Should pass when checking ownership of user", func(t *testing.T) {
		_, oc := setupTest(t)

		ctx := context.Background()
		userId := uuid.MustParse("b805aab0-9533-485f-abc7-f910cfbd50e6")
		userResourceId := uuid.MustParse("b805aab0-9533-485f-abc7-f910cfbd50e6")

		isOwner, err := oc.IsOwner(userId, ResourceTypeUser, userResourceId, ctx)
		if err != nil {
			t.Error("Failed calling IsOwner:", err)
		}
		if isOwner == false {
			t.Error("User should be owner of his profile")
		}

	})

	t.Run("Should fail when checking ownership of user", func(t *testing.T) {
		_, oc := setupTest(t)

		ctx := context.Background()
		userId := uuid.MustParse("b805aab0-9533-485f-abc7-f910cfbd50e6")
		userResourceId := uuid.MustParse("d72d4113-3527-495a-9156-95cf1808f2cb")

		isOwner, err := oc.IsOwner(userId, ResourceTypeUser, userResourceId, ctx)
		if err != nil {
			t.Error("Failed calling IsOwner:", err)
		}
		if isOwner {
			t.Error("User should not be owner of another profile")
		}

	})

	t.Run("Should pass when checking ownership of posts", func(t *testing.T) {
		mock, oc := setupTest(t)

		ctx := context.Background()
		userId := uuid.MustParse("b805aab0-9533-485f-abc7-f910cfbd50e6")
		postResourceId := uuid.MustParse("625ddac2-a366-4dda-82d4-022608b3dd88")

		mock.
			EXPECT().
			GetContent(gomock.Eq(postResourceId), gomock.Any()).
			Return(persistence.Post{
				AuthorID: userId,
			}, nil).
			Times(1)

		isOwner, err := oc.IsOwner(userId, ResourceTypePost, postResourceId, ctx)
		if err != nil {
			t.Error("Failed calling IsOwner:", err)
		}
		if isOwner == false {
			t.Error("User should be owner of his post")
		}

	})

	t.Run("Should fail when checking ownership of posts", func(t *testing.T) {
		mock, oc := setupTest(t)

		ctx := context.Background()
		userId := uuid.MustParse("b805aab0-9533-485f-abc7-f910cfbd50e6")
		postResourceId := uuid.MustParse("c7eb8c5d-7018-401f-bff0-f822932efe2a")
		actualPostAuthorId := uuid.MustParse("d72d4113-3527-495a-9156-95cf1808f2cb")

		mock.
			EXPECT().
			GetContent(gomock.Eq(postResourceId), gomock.Any()).
			Return(persistence.Post{
				AuthorID: actualPostAuthorId,
			}, nil).
			Times(1)

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

var (
	authorIDFirst  = uuid.MustParse("b805aab0-9533-485f-abc7-f910cfbd50e6")
	authorIDSecond = uuid.MustParse("d72d4113-3527-495a-9156-95cf1808f2cb")
	postIDFirst    = uuid.MustParse("625ddac2-a366-4dda-82d4-022608b3dd88")
	postIDSecond   = uuid.MustParse("c7eb8c5d-7018-401f-bff0-f822932efe2a")
)
