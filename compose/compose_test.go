package compose

import (
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/ghttp"
)

type testStruct struct {
	String string         `json:"string"`
	Int    int64          `json:"int"`
	Bool   bool           `json:"bool"`
	Slice  []int          `json:"slice"`
	Map    map[string]int `json:"map"`
}

var testResponse string = `{"string":"test","int":64,"bool":true,"slice":[1,2,3],"map":{"a":1,"b":2,"c":3}}`

var _ = Describe("Compose library", func() {
	Context("creating a new client", func() {
		var (
			u *url.URL
		)

		BeforeEach(func() {
			var err error
			u, err = url.Parse("https://api.compose.io/2016-07")
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should be able to create New() client", func() {
			client, err := New("t35t", u)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(client.Token).To(Equal("t35t"))
		})

		It("should fail to create New() client due to an empty token", func() {
			client, err := New("", u)
			Expect(err).Should(HaveOccurred())
			Expect(client).To(BeNil())
		})

		It("should fail to create New() client due to a nil url.URL", func() {
			client, err := New("t35t", nil)
			Expect(err).Should(HaveOccurred())
			Expect(client).To(BeNil())
		})
	})

	Context("client already created", func() {
		var (
			apiServer *ghttp.Server
			client    *Client
		)

		BeforeEach(func() {
			apiServer = ghttp.NewServer()

			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/test"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, testResponse),
				),
			)
		})

		JustBeforeEach(func() {
			u, err := url.Parse("https://example.com/v1")
			Expect(err).ShouldNot(HaveOccurred())

			client, err = New("test-token-1234567890", u)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("should decodeResponse() correctly", func() {
			t := testStruct{}
			res, err := http.Get("http://example.com/")
			Expect(err).ShouldNot(HaveOccurred())

			err = decodeResponse(res, &t)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(t.Map["a"]).To(Equal(1))
		})
	})
})
