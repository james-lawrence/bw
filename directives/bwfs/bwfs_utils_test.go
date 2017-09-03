package bwfs

import (
	"math/rand"
	"net/url"
	"path/filepath"

	"github.com/icrowley/fake"
)

func fakeURL() string {
	fake.SimplePassword()
	tmp := url.URL{
		Scheme:  randomString("http", "https", "git", "s3"),
		Host:    randomString(fake.DomainName(), fake.IPv4(), fake.IPv6()),
		RawPath: filepath.Join(fake.Word(), fake.Word(), fake.Word(), fake.Word()),
		RawQuery: url.Values{
			fake.Word(): []string{fake.Word()},
		}.Encode(),
	}
	return (&tmp).String()
}

func randomArchive() Archive {
	return Archive{
		URI:   fakeURL(),
		Path:  randomFilepath(5),
		Mode:  randomFilemode(),
		Owner: randomUsername(),
		Group: randomUsername(),
	}
}

func randomFilepath(depth int) string {
	parts := make([]string, 0, depth)
	for i := 0; i < depth; i++ {
		parts = append(parts, fake.CharactersN(5))
	}
	return filepath.Join(parts...)
}

func randomUsername() string {
	return stringFromCharset(
		rand.Intn(30)+1,
		[]rune(CharsetUnixIdent)...,
	)
}

func randomFilemode() uint32 {
	const (
		omask = 7 << (3 * iota) // 00000000000000000000000000000111
		gmask
		umask
		smask
	)

	v := rand.Uint32()
	o := (v & omask)
	g := (v & gmask)
	u := (v & umask)
	// fmt.Printf("\no: %0.32b\nm: %0.32b", o, omask)
	// fmt.Printf("\ng: %0.32b\nm: %0.32b", g, gmask)
	// fmt.Printf("\nu: %0.32b\nm: %0.32b", u, umask)
	// fmt.Printf("\nt: %0.32b\nm: %0.32b\n", u|g|o, omask|gmask|umask)
	// log.Printf("randomFilemode:  %0.32b,%0.4o\n", 0|u|g|o, u|g|o)
	return u | g | o
}

// randomString selects a random string from a set of strings.
func randomString(s ...string) string {
	return s[rand.Int()%len(s)]
}

func stringFromCharset(length int, charset ...rune) string {
	b := make([]rune, length)

	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}

	return string(b)
}
