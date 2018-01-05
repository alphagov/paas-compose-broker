package config_test

import (
	. "github.com/alphagov/paas-compose-broker/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("parsing a whitelist", func() {
	It("parses an empty string as an empty list", func() {
		Expect(ParseIPWhitelist("")).
			To(BeEmpty())
	})

	It("parses a single IP", func() {
		Expect(ParseIPWhitelist("127.0.0.1")).
			To(Equal([]string{"127.0.0.1"}))
	})

	It("parses multiple IPs", func() {
		Expect(ParseIPWhitelist("127.0.0.1,99.99.99.99")).
			To(Equal([]string{"127.0.0.1", "99.99.99.99"}))
	})

	It("returns error for garbage IPs", func() {
		_, err := ParseIPWhitelist("ojnratuh53ggijntboijngk3,0ij90490ti9jo43p;';;1;'")
		Expect(err).To(HaveOccurred())
	})
})
