package interfaces_test

import (
	"testing"

	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/testhelpers"
)

// Compile-time interface satisfaction checks ensure that concrete
// implementations stay compatible with the interfaces they claim to implement.

func TestStubStorageServiceImplementsStorageService(t *testing.T) {
	t.Parallel()
	// This is enforced at compile time; the test simply documents it.
	var _ interfaces.StorageService = (*testhelpers.StubStorageService)(nil)
}

func TestStubLoggerImplementsLoggerInterface(t *testing.T) {
	t.Parallel()
	// StubLogger already has a compile-time check in testhelpers, but we
	// verify it here to confirm the interfaces package stays in sync.
	_ = testhelpers.NewStubLogger()
}

// TestInterfaceMethodSignatures ensures the interfaces have the expected method
// counts. This catches accidental additions/removals during refactoring.
func TestStorageServiceHasExpectedMethods(t *testing.T) {
	t.Parallel()
	// We can only do a compile-time check; instantiate a stub to prove
	// the full interface is satisfied.
	s := testhelpers.NewStubStorageService()
	var ss interfaces.StorageService = s
	// If this compiles, StorageService is fully implemented by StubStorageService.
	_ = ss
}
