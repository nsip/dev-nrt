package reports

import "errors"

//
// set of custome error structures to support qa analysis
// reports
//

var (
	errUnexpectedItemResponse  = errors.New("Item response captured without student completing test")
	errUnexpectedItemScore     = errors.New("Scored test item with status other than AF, P or R")
	errNonZeroItemScore        = errors.New("Non-zero scored test item with status of R")
	errMissingItemScore        = errors.New("Unscored test with status of AF, P or R")
	errMissingItemWritingScore = errors.New("Unscored writing item with status of P or AF")
)

var (
	errUnexpectedAdaptivePathway = errors.New("Adaptive pathway without student undertaking test")
	errUnexpectedScore           = errors.New("Scored test with status other than P or R")
	errRefusedScore              = errors.New("Non-zero score with status of R")
	errMissingScore              = errors.New("Unscored test with status of P or R")
)

var (
	errWritingAdaptive = errors.New("Writing test with adaptive structure")
	errNonAdaptive     = errors.New("Non-writing test with non-adaptive structure")
)

var (
	errNoWritingSubscores      = errors.New("No subscores found for Writing test")
	errUnexpectedItemSubscores = errors.New("Unexpected subscores found for non-writing test item")
)

type itemError struct {
	err              error
	itemRefId        string
	itemResponseJson string
}
