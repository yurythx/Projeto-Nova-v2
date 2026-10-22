package transport

import apperrors "github.com/yurythx/sismarq/internal/domain/errors"

func httputilValidation(msg string) error { return apperrors.Validation(msg) }
