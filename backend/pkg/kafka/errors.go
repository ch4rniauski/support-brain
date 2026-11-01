package kafka

import "errors"

// ErrPermanent marks a non-retryable handler failure (bad payload, validation).
var ErrPermanent = errors.New("permanent")
