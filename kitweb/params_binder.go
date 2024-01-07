package kitweb

import "github.com/kitcat-framework/kitcat/kitweb/httpbind"

var GetParamsBinder = getDefaultParamsBinder

func getDefaultParamsBinder(c *Config) ParamsBinder {
	valueExtractors := append(httpbind.ValuesParamExtractors, c.AdditionalValueExtractors...)
	stringExtractors := append(httpbind.StringsParamExtractors, c.AdditionalStringExtractor...)

	return httpbind.NewBinder(stringExtractors, valueExtractors)
}
