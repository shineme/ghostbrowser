package undetectable

import "os"

// AcquireProfileData shares the launch lock with archive/restore operations.
// The lock lives outside the profile so the directory can be staged atomically.
func AcquireProfileData(dir string) (*os.File, error) { return lockProfile(dir) }
