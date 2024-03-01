package main

import (
	"fmt"
	"github.com/zeebo/errs"
)

// #include "libp2p_definitions.h"
import "C"

var (
	// ErrInvalidHandle is used when the handle passed as an argument is invalid.
	ErrInvalidHandle = errs.Class("invalid handle")
	// ErrNull is returned when an argument is NULL, however it should not be.
	ErrNull = errs.Class("NULL")
	// ErrInvalidArg is returned when the argument is not valid.
	ErrInvalidArg = errs.Class("invalid argument")
)



func mallocError(err error) *C.Libp2pError {
	if err == nil {
		return nil
	}

	cerror := (*C.Libp2pError)(calloc(1, C.sizeof_Libp2pError))
	cerror.message = C.CString(fmt.Sprintf("%+v", err))

	cerror.code = 0; //TODO determine error code
	return cerror;
}

func mallocErrorRaw(code int32, message string) *C.Libp2pError {
	cerror := (*C.Libp2pError)(calloc(1, C.sizeof_Libp2pError))
	cerror.message = C.CString(message)
	cerror.code = C.int(code);
	return cerror;
}