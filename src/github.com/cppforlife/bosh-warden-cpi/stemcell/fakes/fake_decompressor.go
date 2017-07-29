package fakes

type FakeDecompressor struct {
	DecompressSrcForCall []string
	DecompressDstForCall []string

	DecompressError error
}

func (f *FakeDecompressor) Decompress(src, dst string) error {
	f.DecompressSrcForCall = append(f.DecompressSrcForCall, src)
	f.DecompressDstForCall = append(f.DecompressDstForCall, dst)

	return f.DecompressError
}
