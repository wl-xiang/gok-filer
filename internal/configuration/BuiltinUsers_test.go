package configuration

import (
	"testing"

	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/test"
	"github.com/forceu/gokapi/internal/test/testconfiguration"
)

// TestInitializeBuiltinUsers verifies that the built-in accounts are created exactly once,
// stored with a hashed password and that existing users are never overwritten.
func TestInitializeBuiltinUsers(t *testing.T) {
	// The shared TestMain has already written the test databases, but closed the connection again
	testconfiguration.Create(false)
	Load()
	ConnectDatabase()

	InitializeBuiltinUsers()

	// The administrator account exists, has admin rights (but is not the super admin) and
	// stores the password as a hash
	admin, ok := database.GetUserByName(BuiltinAdminName)
	test.IsEqualBool(t, ok, true)
	test.IsEqualBool(t, admin.IsAdmin(), true)
	test.IsEqualBool(t, admin.IsSuperAdmin(), false)
	test.IsEqualBool(t, admin.Password != BuiltinAdminPassword, true)
	validAdmin, _ := VerifyPassword(BuiltinAdminPassword, admin.Password, "")
	test.IsEqualBool(t, validAdmin, true)

	// The test configuration already contains a user called "user", therefore the built-in
	// initialisation must not overwrite the existing entry
	existingBefore, ok := database.GetUserByName(BuiltinUserName)
	test.IsEqualBool(t, ok, true)
	InitializeBuiltinUsers()
	existingAfter, ok := database.GetUserByName(BuiltinUserName)
	test.IsEqualBool(t, ok, true)
	test.IsEqualString(t, existingAfter.Password, existingBefore.Password)
	test.IsEqualString(t, existingAfter.Name, existingBefore.Name)

	// Repeated startups must not create additional entries
	userCount := len(database.GetAllUsers())
	InitializeBuiltinUsers()
	test.IsEqualInt(t, len(database.GetAllUsers()), userCount)

	// When the feature is disabled, no account is created
	database.DeleteUser(admin.Id)
	parsedEnvironment.CreateBuiltinUsers = false
	InitializeBuiltinUsers()
	_, ok = database.GetUserByName(BuiltinAdminName)
	test.IsEqualBool(t, ok, false)
	parsedEnvironment.CreateBuiltinUsers = true
}
