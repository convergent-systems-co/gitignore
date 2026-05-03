package source

import "fmt"

// SourceFactory builds a Source from raw configuration parameters. Factories
// receive every config knob the standard sources care about; ignore the
// arguments your source doesn't need.
type SourceFactory func(localPath, templateURL string) (Source, error)

// registry holds the named factories that NewSourceManager can build.
// Pre-populated with the three built-in sources; extended via RegisterSource
// from an init() in another file or a downstream package.
var registry = map[string]SourceFactory{
	NameLocal: func(localPath, _ string) (Source, error) {
		return NewLocalSourceWithDir(localPath), nil
	},
	NameGitHub: func(_, templateURL string) (Source, error) {
		return NewGitHubSource(templateURL)
	},
	NameToptal: func(_, _ string) (Source, error) {
		return NewToptalSource(), nil
	},
}

// RegisterSource adds or replaces a factory in the registry. Intended for
// init() in packages that introduce a new Source type.
func RegisterSource(name string, f SourceFactory) {
	registry[name] = f
}

// buildSource invokes the registered factory for name.
func buildSource(name, localPath, templateURL string) (Source, error) {
	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("no source registered for %q", name)
	}
	return f(localPath, templateURL)
}
