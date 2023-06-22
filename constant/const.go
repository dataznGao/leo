package constant

const Separator = "/"

const LogContent = "\"this is a log\""

type BingoFaultType int

const (
	ValueFault BingoFaultType = iota
	NullFault
	ExceptionShortcircuitFault
	ExceptionUncaughtFault
	ExceptionUnhandledFault
	AttributeShadowedFault
	AttributeReversoFault
	SwitchMissDefaultFault
	ConditionBorderFault
	ConditionInversedFault
	SyncFault
)
