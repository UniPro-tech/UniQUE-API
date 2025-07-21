package errorresponse

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const UnauthorizedCode = 1001
const UnauthorizedMessage = "Unauthorized access"

var UnauthorizedError = ErrorResponse{
	Code:    UnauthorizedCode,
	Message: UnauthorizedMessage,
}

const InvalidTokenCode = 1002
const InvalidTokenMessage = "Invalid token"

var InvalidTokenError = ErrorResponse{
	Code:    InvalidTokenCode,
	Message: InvalidTokenMessage,
}

const ExpiredTokenCode = 1003
const ExpiredTokenMessage = "Expired token"

var ExpiredTokenError = ErrorResponse{
	Code:    ExpiredTokenCode,
	Message: ExpiredTokenMessage,
}

const ForbiddenCode = 1101
const ForbiddenMessage = "Forbidden access"

var ForbiddenError = ErrorResponse{
	Code:    ForbiddenCode,
	Message: ForbiddenMessage,
}

const MissingRequestCode = 2001
const MissingRequestMessage = "Missing required fields"

var MissingRequestError = ErrorResponse{
	Code:    MissingRequestCode,
	Message: MissingRequestMessage,
}

const InvalidFormatCode = 2002
const InvalidFormatMessage = "Invalid format"

var InvalidFormatError = ErrorResponse{
	Code:    InvalidFormatCode,
	Message: InvalidFormatMessage,
}

const TooShortCode = 2003
const TooShortMessage = "Input is too short"

var TooShortError = ErrorResponse{
	Code:    TooShortCode,
	Message: TooShortMessage,
}

const TooLongCode = 2004
const TooLongMessage = "Input is too long"

var TooLongError = ErrorResponse{
	Code:    TooLongCode,
	Message: TooLongMessage,
}

const OutOfRangeCode = 2005
const OutOfRangeMessage = "Input is out of range"

var OutOfRangeError = ErrorResponse{
	Code:    OutOfRangeCode,
	Message: OutOfRangeMessage,
}

const NotAllowedValueCode = 2006
const NotAllowedValueMessage = "Not allowed value"

var NotAllowedValueError = ErrorResponse{
	Code:    NotAllowedValueCode,
	Message: NotAllowedValueMessage,
}

const MissmatchedPatternCode = 2007
const MissmatchedPatternMessage = "Input does not match the required pattern"

var MissmatchedPatternError = ErrorResponse{
	Code:    MissmatchedPatternCode,
	Message: MissmatchedPatternMessage,
}

const InvalidRequestCode = 2008
const InvalidRequestMessage = "Invalid request"

var InvalidRequestError = ErrorResponse{
	Code:    InvalidRequestCode,
	Message: InvalidRequestMessage,
}

const NotFoundCode = 3001
const NotFoundMessage = "Resource not found"

var NotFoundError = ErrorResponse{
	Code:    NotFoundCode,
	Message: NotFoundMessage,
}

const AlreadyExistsCode = 3002
const AlreadyExistsMessage = "Resource already exists"

var AlreadyExistsError = ErrorResponse{
	Code:    AlreadyExistsCode,
	Message: AlreadyExistsMessage,
}

const InternalServerErrorMessage = "Internal server error"
const DBErrorCode = 4001

var DBError = ErrorResponse{
	Code:    DBErrorCode,
	Message: InternalServerErrorMessage,
}

const InternalServerErrorCode = 4002

var InternalServerError = ErrorResponse{
	Code:    InternalServerErrorCode,
	Message: InternalServerErrorMessage,
}

const ExternalServiceUnreachableCode = 5001
const ExternalServiceUnreachableMessage = "External service is unreachable"

var ExternalServiceUnreachableError = ErrorResponse{
	Code:    ExternalServiceUnreachableCode,
	Message: ExternalServiceUnreachableMessage,
}

const ExternalServiceTimeoutCode = 5002
const ExternalServiceTimeoutMessage = "External service request timed out"

var ExternalServiceTimeoutError = ErrorResponse{
	Code:    ExternalServiceTimeoutCode,
	Message: ExternalServiceTimeoutMessage,
}

const InvalidExternalServiceResponseCode = 5003
const InvalidExternalServiceResponseMessage = "Invalid response from external service"

var InvalidExternalServiceResponseError = ErrorResponse{
	Code:    InvalidExternalServiceResponseCode,
	Message: InvalidExternalServiceResponseMessage,
}

const ExternalServiceAuthenticationErrorCode = 5004
const ExternalServiceAuthenticationErrorMessage = "Authentication error with external service"

var ExternalServiceAuthenticationError = ErrorResponse{
	Code:    ExternalServiceAuthenticationErrorCode,
	Message: ExternalServiceAuthenticationErrorMessage,
}

const ExternalStorageErrorCode = 5005
const ExternalStorageErrorMessage = "External storage error"

var ExternalStorageError = ErrorResponse{
	Code:    ExternalStorageErrorCode,
	Message: ExternalStorageErrorMessage,
}

// キャッチはできているけど未分類なエラー
const UncategorizedErrorCode = 9001
const UncategorizedErrorMessage = "An error occurred"

var UncategorizedError = ErrorResponse{
	Code:    UncategorizedErrorCode,
	Message: UncategorizedErrorMessage,
}

const FeatureNotImplementedCode = 9002
const FeatureNotImplementedMessage = "Feature not implemented"

var FeatureNotImplementedError = ErrorResponse{
	Code:    FeatureNotImplementedCode,
	Message: FeatureNotImplementedMessage,
}

// こっちは完全に未知。ハンドルすらできていない
const UnknownErrorCode = 9003
const UnknownErrorMessage = "Unknown error occurred. Please contact support."

var UnknownError = ErrorResponse{
	Code:    UnknownErrorCode,
	Message: UnknownErrorMessage,
}

const LogicErrorCode = 9004
const LogicErrorMessage = "Unexpected situation encountered. Please contact support."

var LogicError = ErrorResponse{
	Code:    LogicErrorCode,
	Message: LogicErrorMessage,
}

const UnhandledErrorCode = 9099
const UnhandledErrorMessage = "Unhandled error occurred. Please contact support."

var UnhandledError = ErrorResponse{
	Code:    UnhandledErrorCode,
	Message: UnhandledErrorMessage,
}
