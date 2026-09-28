package configuration

/**
Initialisation of the built-in system accounts.

Two accounts are provided out of the box:
  - admin / admin1234  -> administrator with all permissions
  - user  / user1234   -> guest account with read-only permissions

The accounts are created once on startup if they do not exist yet. Existing users with the
same name are never touched, so no duplicate entries are created and changed passwords are
preserved. Passwords are stored as Argon2id hashes, never in plain text.
*/

import (
	"fmt"

	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/models"
)

// Credentials of the built-in accounts. They are only used the first time the accounts are
// created, afterwards the hashed values from the database are used.
const (
	// BuiltinAdminName is the username of the built-in administrator
	BuiltinAdminName = "admin"
	// BuiltinAdminPassword is the initial password of the built-in administrator
	BuiltinAdminPassword = "admin1234"
	// BuiltinUserName is the username of the built-in read-only guest account
	BuiltinUserName = "user"
	// BuiltinUserPassword is the initial password of the built-in read-only guest account
	BuiltinUserPassword = "user1234"
)

// InitializeBuiltinUsers creates the built-in accounts if they are enabled and do not exist yet.
// It is idempotent: calling it multiple times (e.g. on every startup) will not create duplicates.
func InitializeBuiltinUsers() {
	if !parsedEnvironment.CreateBuiltinUsers {
		return
	}
	createdAdmin := createBuiltinUserIfMissing(BuiltinAdminName, BuiltinAdminPassword,
		models.UserLevelAdmin, models.UserPermissionAll)
	createdUser := createBuiltinUserIfMissing(BuiltinUserName, BuiltinUserPassword,
		models.UserLevelUser, models.UserPermissionNone)
	if createdAdmin || createdUser {
		fmt.Println("Built-in accounts are available: " + BuiltinAdminName + " (administrator), " +
			BuiltinUserName + " (read-only guest)")
	}
}

// createBuiltinUserIfMissing creates a user with the given name if no user with that name exists
// yet. It returns true if a new user was created.
func createBuiltinUserIfMissing(name, password string, level models.UserRank, permissions models.UserPermission) bool {
	if _, exists := database.GetUserByName(name); exists {
		return false
	}
	user := models.User{
		Name:        name,
		Permissions: permissions,
		UserLevel:   level,
		Password:    HashPassword(password, false, ""),
	}
	database.SaveUser(user, true)
	fmt.Println("Created built-in user: " + name)
	return true
}
