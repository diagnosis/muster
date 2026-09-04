
//e2e/fixtures/copy.ts

export const EMAIL_EXISTS = 'email already registered'
export const SHORT_PASSWORD = 'Password must be at least 8 characters'
export const INVALID_PASSWORD = 'Password must contain at least one uppercase letter, one lowercase letter, and one number'
export const INVALID_LOGIN = 'invalid login credentials'
export const VALIDATION_FAILED = 'validation failed'
export const ACCEPTED = "You're going! 🎉"
export const DECLINED ='The host declined this request.'
export const CANCELLED = 'This outing was cancelled'
export const CAPACITY_WARNING = "Looks like there isn't room for 2 right now."
export const ACCEPT_409 = 'outing is full or request is not pending'
export const REQUESTED = 'Requested — waiting on host'
export const FULL_WARNING = '⚠️This outing is full — you can still request in case a spot opens.'
export const OUTING_FULL = 'This outing is full.'
export const NO_SEAT_LEFT = 'No seats left — a driver could open more spots.'
export const SEAT_SHORT = '1 more seats needed — join as a driver?'
export const MAX_SIZE_SHRINK_CONFLICT = 'cannot reduce group size below current committed members'
export const REMOVED = 'The host removed you from this outing.'
export const REMOVE_CONFIRMATION = 'This removes them from the roster and frees their seats. They won\'t be able to request again.'
export const REMOVE_BTN_TEXT = 'Yes, remove'
export const REMOVE_ARIA = 'Remove'

export const INVALID_NUMBER_FIELDS = 'Check the number fields — something is invalid.'

export const INVALID_MAX_SIZE = 'Max size must be a whole number of at least 2.'
export const INVALID_HOST_SEATS = 'Host seats must be a whole number, 0 or more.'
export const INVALID_COST = 'Cost per seat must be a positive amount, up to two decimals.'


// verify email page
export const VERIFY_SUCCESS = 'Your email is verified.'
export const VERIFY_SUCCESS_CTA = 'Log in'
export const VERIFY_EXPIRED_OR_USED = 'This link has expired or was already used.'
export const VERIFY_INVALID = 'This link is not valid.'
export const VERIFY_PENDING = 'Verifying…'

// login gate
export const EMAIL_NEED_VERIFICATION = 'Email needs to be verified.'
export const RESEND_RESPONSE = 'If email provided matches our record, you will receive a verification email'