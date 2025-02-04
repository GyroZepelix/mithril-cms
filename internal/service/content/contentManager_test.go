package content

import (
	"context"
	"database/sql"
	"testing"

	"github.com/GyroZepelix/mithril-cms/internal/errs"
	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
	mock_persistence "github.com/GyroZepelix/mithril-cms/internal/storage/persistence/mock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var UUID_1 = uuid.MustParse("6b1b090e-0ed3-425b-a6d5-5daa8de1ce6f")
var UUID_2 = uuid.MustParse("31351636-0ed3-425b-a6d5-5daa8de1ce6f")

func setupTest(t *testing.T) (*mock_persistence.MockQuerier, Manager, context.Context) {

	ctrl := gomock.NewController(t)
	mock := mock_persistence.NewMockQuerier(ctrl)
	manager := NewManager(mock)
	ctx := context.Background()
	return mock, manager, ctx
}

func TestGetContent(t *testing.T) {

	mock, cm, ctx := setupTest(t)

	post := persistence.Post{
		ID:    UUID_1,
		Title: "title 1",
	}

	t.Run("Should get content successfuly", func(t *testing.T) {
		t.Parallel()
		mock.EXPECT().
			GetContent(gomock.Any(), gomock.Eq(UUID_1)).
			Return(post, nil).
			Times(1)

		gotPost, err := cm.GetContent(UUID_1, ctx)

		assert.NoError(t, err)
		assert.Equal(t, post, gotPost)
	})

	t.Run("Should not find content with given UUID", func(t *testing.T) {
		t.Parallel()
		mock.EXPECT().
			GetContent(gomock.Any(), gomock.Eq(UUID_1)).
			Return(persistence.Post{}, sql.ErrNoRows).
			Times(1)

		_, err := cm.GetContent(UUID_1, ctx)

		if assert.Error(t, err) {
			assert.Equal(t, errs.ErrNotFound, err)
		}
	})

}

func TestListContents(t *testing.T) {

	t.Parallel()
	mock, cm, ctx := setupTest(t)

	post1 := persistence.Post{
		ID:    UUID_1,
		Title: "title 1",
	}
	post2 := persistence.Post{
		ID:    UUID_2,
		Title: "title 2",
	}
	posts := []persistence.Post{post1, post2}

	mock.EXPECT().
		ListContents(gomock.Any()).
		Return(posts, nil).
		Times(1)

	gotPosts, err := cm.ListContents(ctx)

	assert.NoError(t, err)
	assert.Equal(t, posts, gotPosts)

}
