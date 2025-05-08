package content

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/GyroZepelix/mithril-cms/internal/errs"
	"github.com/GyroZepelix/mithril-cms/internal/logging"
	"github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
	mock_persistence "github.com/GyroZepelix/mithril-cms/internal/storage/persistence/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var POST_UUID_1 = uuid.MustParse("6b1b090e-0ed3-425b-a6d5-5daa8de1ce6f")
var POST_UUID_2 = uuid.MustParse("31351636-0ed3-425b-a6d5-5daa8de1ce6f")
var USER_UUID = uuid.MustParse("13507239-0ed3-425b-a6d5-5daa8de1ce6f")

func setupTest(t *testing.T) (*mock_persistence.MockQuerier, Manager, context.Context) {
	ctrl := gomock.NewController(t)
	querierDbMock := mock_persistence.NewMockQuerier(ctrl)
	manager := NewManager(querierDbMock)
	ctx := context.Background()
	logging.Init(os.Stdout)
	return querierDbMock, manager, ctx
}

func TestGetContent(t *testing.T) {

	mock, cm, ctx := setupTest(t)

	post := persistence.Post{
		ID:    POST_UUID_1,
		Title: "title 1",
	}

	t.Run("Should get content successfuly", func(t *testing.T) {
		t.Parallel()
		mock.EXPECT().
			GetContent(gomock.Any(), gomock.Eq(POST_UUID_1)).
			Return(post, nil).
			Times(1)

		gotPost, err := cm.GetContent(POST_UUID_1, ctx)

		assert.NoError(t, err)
		assert.Equal(t, post, gotPost)
	})

	t.Run("Should not find content with given UUID", func(t *testing.T) {
		t.Parallel()
		mock.EXPECT().
			GetContent(gomock.Any(), gomock.Eq(POST_UUID_1)).
			Return(persistence.Post{}, sql.ErrNoRows).
			Times(1)

		_, err := cm.GetContent(POST_UUID_1, ctx)

		if assert.Error(t, err) {
			assert.Equal(t, errs.ErrNotFound, err)
		}
	})

}

func TestListContents(t *testing.T) {

	t.Parallel()
	mock, contentManager, ctx := setupTest(t)

	post1 := persistence.PostView{
		ID:    POST_UUID_1,
		Title: "title 1",
	}
	post2 := persistence.PostView{
		ID:    POST_UUID_2,
		Title: "title 2",
	}
	posts := []persistence.PostView{post1, post2}

	mock.EXPECT().
		ListContentsWithCategories(gomock.Any()).
		Return(posts, nil).
		Times(1)

	gotPosts, err := contentManager.ListContents(ctx)

	assert.NoError(t, err)
	assert.Equal(t, posts, gotPosts)

}

func TestCreateContent(t *testing.T) {

	mock, cm, ctx := setupTest(t)

	postParam := persistence.CreateContentParams{
		Title:    "Is Bread GOOD?",
		Slug:     "is-bread-good",
		Content:  "Lorem Ipsum",
		AuthorID: USER_UUID,
	}
	post := persistence.Post{
		ID:       POST_UUID_1,
		Title:    "Is Bread GOOD?",
		Slug:     "is-bread-good",
		Content:  "Lorem Ipsum",
		AuthorID: USER_UUID,
		Status:   "draft",
	}

	t.Run("Should create content successfuly", func(t *testing.T) {
		t.Parallel()
		mock.EXPECT().
			CreateContent(gomock.Any(), gomock.Eq(postParam)).
			Return(post, nil).
			Times(1)

		gotPost, err := cm.CreateContent(postParam.Title, postParam.Content, USER_UUID, ctx)

		assert.NoError(t, err)
		assert.Equal(t, post, *gotPost)
	})

	t.Run("Should fail to create content", func(t *testing.T) {
		t.Parallel()
		mockError := errors.New("MOCK_ERROR")
		mock.EXPECT().
			CreateContent(gomock.Any(), gomock.Any()).
			Return(persistence.Post{}, mockError).
			Times(1)

		_, err := cm.CreateContent(postParam.Title, postParam.Content, USER_UUID, ctx)

		if assert.Error(t, err) {
			assert.Equal(t, mockError, err)
		}
	})

}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic title",
			input:    "Is white rice good?",
			expected: "is-white-rice-good",
		},
		{
			name:     "Title with exclamation",
			input:    "Birds spotted in the park!",
			expected: "birds-spotted-in-the-park",
		},
		{
			name:     "Simple title",
			input:    "How to create Bread",
			expected: "how-to-create-bread",
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := generateSlug(tt.input)
			if got != tt.expected {
				t.Errorf("generateSlug() = %v, want %v", got, tt.expected)
			}
		})
	}

}
