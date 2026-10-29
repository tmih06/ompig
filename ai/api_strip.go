package ai

// StrippableAPIs lists the APIs a Ompiglet can strip (strip.apis). A Ompiglet
// Binary compiles each one's implementation out with the build tag
// ompig_strip_<api> (see internal/pigstrip.Tag), and a Ompiglet that Stock Ompig
// runs disables it at runtime. A stripped API's models are not offered
// (OfferedModels), and building a provider for one of them fails.
// ompig additive (D92): Stock Ompig strips no API.
func StrippableAPIs() []API {
	return []API{APIBedrockConverseStream, APIGoogleVertex, APIMistralConversations}
}
