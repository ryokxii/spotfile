//go:build darwin

package trash

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>
#include <stdlib.h>

// trashItem moves path to the user's Trash. It returns NULL on success, or a
// malloc'd error message the caller frees.
static char *trashItem(const char *path) {
	@autoreleasepool {
		NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
		NSError *error = nil;
		if ([[NSFileManager defaultManager] trashItemAtURL:url resultingItemURL:nil error:&error]) {
			return NULL;
		}
		NSString *message = error.localizedDescription ?: @"unknown error";
		return strdup(message.UTF8String);
	}
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

// Trash moves path to the Trash through NSFileManager, which needs no
// permission prompt and lets the user put the file back from Finder.
func (System) Trash(path string) error {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	if msg := C.trashItem(cpath); msg != nil {
		defer C.free(unsafe.Pointer(msg))
		return errors.New(C.GoString(msg))
	}
	return nil
}
