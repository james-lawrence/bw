package dialers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("parseURI", func() {
	It("should extract the procol name and address", func() {
		proto, host, err := parseURI("bw.agent://QmNMbFdN9s7R1PBr9VqZqVrsZzunwh61hD1vTR96AmohGX")
		Expect(err).To(Succeed())
		Expect(string(proto)).To(Equal("bw.agent"))
		Expect(host).To(Equal("QmNMbFdN9s7R1PBr9VqZqVrsZzunwh61hD1vTR96AmohGX"))
	})
})
