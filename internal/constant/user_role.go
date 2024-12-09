package constant

type UserRole int

const (
	UserRoleReader UserRole = iota
	UserRoleAuthor
	UserRoleEditor
	UserRoleAdmin
)

var UserRoleName = map[UserRole]string{
	UserRoleReader: "reader",
	UserRoleAuthor: "author",
	UserRoleEditor: "editor",
	UserRoleAdmin:  "admin",
}
