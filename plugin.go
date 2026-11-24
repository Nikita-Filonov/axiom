package axiom

// Plugin configures a test Config before execution, for example by adding
// hooks, runtime wraps, or sinks. Runner plugins run before Case plugins.
// Plugins run once on a planning Config and again on each attempt Config;
// this includes skipped and excluded attempts. Installation must be free of
// execution side effects; register wrappers or sinks for runtime work.
type Plugin func(cfg *Config)
