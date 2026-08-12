package phash

// Decode registration: NewFromPath loads image files through image.Decode,
// which only recognizes a format once its decoder is registered. The package
// blank-imports the decoders its file-loading API needs so callers never have
// to, mirroring Image::PHash 0.3's Imager, which loads JPEGs out of the box.
// The checked-in _examples fixtures are JPEGs, so image/jpeg is registered
// here.
import _ "image/jpeg"
