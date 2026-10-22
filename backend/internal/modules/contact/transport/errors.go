package transport

import apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"

func httputilValidation(msg string) error { return apperrors.Validation(msg) }
