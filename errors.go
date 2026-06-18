package brdoc

import "errors"

// Sentinel errors. Compare with errors.Is / errors.As; wrap with %w when adding context.
var (
	// ErrInvalidLength indicates the document does not have the expected number of characters.
	ErrInvalidLength = errors.New("brdoc: invalid document length")
	// ErrInvalidFormat indicates the document does not match the expected shape.
	ErrInvalidFormat = errors.New("brdoc: invalid document format")
	// ErrUnknownKind indicates a Kind that is not registered.
	ErrUnknownKind = errors.New("brdoc: unknown document kind")
	// ErrUnsupported indicates an operation is not supported for the given Kind.
	ErrUnsupported = errors.New("brdoc: operation not supported for this kind")
	// ErrUFNotImplemented indicates the requested federative unit has no implementation yet.
	ErrUFNotImplemented = errors.New("brdoc: federative unit not implemented")
)
