package axiom

// Plugin configures a test Config before execution, for example by adding
// hooks, runtime wraps, or sinks. Runner plugins run before Case plugins.
// Plugins run once on a planning Config and again on each attempt Config;
// a skipped case may have no attempt Config.
type Plugin func(cfg *Config)
