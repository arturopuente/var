package ui

// FileSelectedMsg is sent when a file is selected in the sidebar
type FileSelectedMsg struct {
	Path string
}

// DiffLoadedMsg is sent when diff content has been loaded
type DiffLoadedMsg struct {
	Content string
	Path    string
}

// CommitChangedMsg is sent when navigating commit history
type CommitChangedMsg struct {
	Index int
	Hash  string
}

// ErrorMsg is sent when an error occurs
type ErrorMsg struct {
	Err error
}
