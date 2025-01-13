package permission

import (
	"context"

	"github.com/GyroZepelix/mithril-cms/internal/service/content"
	"github.com/google/uuid"
)

type OwnershipChecker interface {
	IsOwner(userID uuid.UUID, resourceType ResourceType, resourceID uuid.UUID, ctx context.Context) (bool, error)
}

func NewOwnershipChecker(contentManager content.Manager) OwnershipChecker {
	return &OwnershipService{
		contentManager,
	}
}

type OwnershipService struct {
	contentManager content.Manager
}

// IsOwner checks if a user is the owner of a specified resource.
//
// Parameters:
//   - userID: The ID of the user to check ownership for.
//   - resourceType: The type of resource (e.g., ResourceTypeUser, ResourceTypePost).
//   - resourceID: The ID of the resource to check ownership of.
//   - ctx: The context for the operation, which may be used for cancellation or passing values.
//
// Returns:
//   - A boolean indicating whether the user is the owner of the resource.
//   - An error if any issues occur during the ownership check.
//
// Note: For ResourceTypePost, the method assumes that resourceID is a string
// representation of an integer and will return false if parsing fails.
func (o *OwnershipService) IsOwner(userID uuid.UUID, resourceType ResourceType, resourceID uuid.UUID, ctx context.Context) (bool, error) {
	switch resourceType {
	case ResourceTypeUser:
		if resourceID == userID {
			return true, nil
		}
		return false, nil
	case ResourceTypePost:
		post, err := o.contentManager.GetContent(resourceID, ctx)
		if err != nil {
			return false, err
		}

		postAuthorId := post.AuthorID
		if postAuthorId == userID {
			return true, nil
		}
		return false, nil
	case ResourceTypeComment:
		return false, nil
	default:
		return false, nil
	}
}
