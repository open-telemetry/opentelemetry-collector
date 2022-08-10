What is this new component httpprovider?
- An implementation of `confmap.Provider` for HTTP (httpprovider) allows OTEL Collector the ability to load configuration for itself by fetching and reading config files stored in HTTP servers.

How this new component httpprovider works?
- It will be called by `confmap.Resolver` to load configurations for OTEL Collector.
- By giving a config URI starting with prefix 'http://', this httpprovider will be used to download config files from given HTTP URIs, and then used the downloaded config files to deploy the OTEL Collector.
- In our code, we check the validity scheme and string pattern of HTTP URIs. And also check if there are any problems on config downloading and config deserialization.

Expected URI format:
- http://...

Prerequistes:
- Need to setup a HTTP server ahead, which returns with a config files according to the given URI