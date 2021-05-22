package rsax

import (
	"encoding/hex"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

// these tests are just to test the functionality, not the actual randomness.
var _ = Describe("NewSHA512CSPRNG", func() {
	DescribeTable("generate data", func(seed string, bits int, expected string) {
		rand := NewSHA512CSPRNG([]byte(seed))
		b := make([]byte, bits)
		n, err := rand.Read(b)
		Expect(err).To(Succeed())
		Expect(n).To(Equal(bits))
		// fmt.Println("output", bits, hex.EncodeToString(b))
		Expect(hex.EncodeToString(b)).To(Equal(expected))
	},
		Entry("example 1",
			"helloworld",
			128,
			"1594244d52f2d8c12b142bb61f47bc2eaf503d6d9ca8480cae9fcf112f66e4967dc5e8fa98285e36db8af1b8ffa8b84cb15e0fbcf836c3deb803c13f37659a604adae9e959b0f671c39fd27164861552735d51f6002818a89b9b17ff93d220033f7642a71094b329c9ee973fdb8127a39434bfec967614fe03ed4159cd86ee8f",
		),
		Entry("example 2",
			"helloworld",
			256,
			"1594244d52f2d8c12b142bb61f47bc2eaf503d6d9ca8480cae9fcf112f66e4967dc5e8fa98285e36db8af1b8ffa8b84cb15e0fbcf836c3deb803c13f37659a604adae9e959b0f671c39fd27164861552735d51f6002818a89b9b17ff93d220033f7642a71094b329c9ee973fdb8127a39434bfec967614fe03ed4159cd86ee8f4f1cbfe18080d03ade7fe2311d4eedb7502ea2be6800deab1ddc31359c0a05983e34e50d100385580acd36e9a2442bd5ff4a42c2d58de213fa01f5d2abbd23ebd8bd8d34261f98a4156a8d183511249b865dc4ecb2ec118dc36f6c97e4f8ffc7ecebd6be18e33a424a193fd459a20aae1186eb1f27ec6d9537d5d861928422fc",
		),
		Entry("example 3",
			"helloworld",
			512,
			"1594244d52f2d8c12b142bb61f47bc2eaf503d6d9ca8480cae9fcf112f66e4967dc5e8fa98285e36db8af1b8ffa8b84cb15e0fbcf836c3deb803c13f37659a604adae9e959b0f671c39fd27164861552735d51f6002818a89b9b17ff93d220033f7642a71094b329c9ee973fdb8127a39434bfec967614fe03ed4159cd86ee8f4f1cbfe18080d03ade7fe2311d4eedb7502ea2be6800deab1ddc31359c0a05983e34e50d100385580acd36e9a2442bd5ff4a42c2d58de213fa01f5d2abbd23ebd8bd8d34261f98a4156a8d183511249b865dc4ecb2ec118dc36f6c97e4f8ffc7ecebd6be18e33a424a193fd459a20aae1186eb1f27ec6d9537d5d861928422fc3cceb3f62e0063d201da9c8e4f7e369b896fcf26e2aa517de3984ee145e92bbc083346695da3509cbc246b1e83605ada8862e664f0772399bb9c16b35b39068ebd075c017f6b8d293e4829b28ada742a02773308e9c9edd8bc98648d9cc88f977568582c15165b7b0b03363b946bd2c921f040300774e3db638e029a80a79e8de1fb96e82374669ed2eb274005a929a14b9387c86cf896de86ca8c0494e9655da685ef80008dffcfdfe4e2e093a9eccaa2059228a9afda19c7913a3bae1a4b6940bf7e628a45ee0c79a947ad3767c6aca93127530e3e07ae84b68d8ceedcb771730c23875caef8d5be66cfbf1f8bc9348a60a0a3531cfe6b6da88bf9041d0def",
		),
		Entry("example 4",
			"testing",
			128,
			"521b9ccefbcd14d179e7a1bb877752870a6d620938b28a66a107eac6e6805b9d0989f45b5730508041aa5e710847d439ea74cd312c9355f1f2dae08d40e41d50e5e50e3f270de5e55b7904f5116e5d28ea98cbf0df6bc4dd64499873f115fbba806829b10f513e2a259005421cf2215fa62066cd859238016d7798a37c9e6d02",
		),
	)
})
