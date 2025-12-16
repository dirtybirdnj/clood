package parser

// Package parser provides language-specific source code parsing.
// Used for symbol extraction, dependency analysis, etc.

// Language represents a supported programming language
type Language string

const (
	LangGo         Language = "go"
	LangPython     Language = "python"
	LangJavaScript Language = "javascript"
	LangTypeScript Language = "typescript"
	LangRust       Language = "rust"
	LangUnknown    Language = "unknown"
)

// DetectLanguage determines the language from a file extension
func DetectLanguage(ext string) Language {
	switch ext {
	case ".go":
		return LangGo
	case ".py":
		return LangPython
	case ".js", ".jsx":
		return LangJavaScript
	case ".ts", ".tsx":
		return LangTypeScript
	case ".rs":
		return LangRust
	default:
		return LangUnknown
	}
}
