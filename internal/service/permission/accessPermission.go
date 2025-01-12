package permission

type ResourceType string

const (
	ResourceTypePost    ResourceType = "post"
	ResourceTypeComment ResourceType = "comment"
	ResourceTypeUser    ResourceType = "user"
)

type Permission uint8

const (
	CanCreate Permission = iota
	CanRead
	CanUpdate
	CanDelete
)

type PermissionLevel uint8

const (
	Owned PermissionLevel = iota
	All
)

type AccessPermission struct {
	ResourceType    ResourceType
	Permission      Permission
	PermissionLevel PermissionLevel
}
