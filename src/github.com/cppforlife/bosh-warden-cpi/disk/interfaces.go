package disk

type Creator interface {
	Create(size int) (Disk, error)
}

type Finder interface {
	Find(id string) (Disk, error)
}

type Disk interface {
	ID() string
	Path() string

	Exists() (bool, error)
	Delete() error
}
