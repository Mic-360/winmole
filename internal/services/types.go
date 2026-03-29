package services

type OperationReport struct {
	Title   string
	Count   int
	Bytes   int64
	Message string
	Errors  []string
}
