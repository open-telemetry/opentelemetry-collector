package service

import (
	"fmt"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/service/internal/configunmarshaler"
)

func printMarshalErrors(exterrs []configunmarshaler.ExtensionValidateError, experrs []configunmarshaler.ExporterValidateError, recerrs []configunmarshaler.ReceiverValidateError, procerrs []configunmarshaler.ProcessorValidateError) {
	count := 1
	for _, exterr := range exterrs {
		validateErrorText(count, exterr.Component, exterr.Id, exterr.Err)
		count++
	}
	for _, experr := range experrs {
		validateErrorText(count, experr.Component, experr.Id, experr.Err)
		count++
	}
	for _, recerr := range recerrs {
		validateErrorText(count, recerr.Component, recerr.Id, recerr.Err)
		count++
	}
	for _, procerr := range procerrs {
		validateErrorText(count, procerr.Component, procerr.Id, procerr.Err)
		count++
	}
}

func validateErrorText(count int, idType string, id config.ComponentID, err string) {
	fmt.Printf("Error %d\n", count)
	fmt.Printf("=============\n")
	fmt.Printf("%s:\n", idType)
	fmt.Printf("	%s\n", id)
	fmt.Printf("	^---^--- %s\n", err)
}
