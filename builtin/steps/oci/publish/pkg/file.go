package pkg

type File struct {
	From string
	To   string
}

func NewFile(from, to string) *File {
	return &File{
		From: from,
		To:   to,
	}
}
