package pagewriter

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("SignIn Page", func() {

	Context("SignIn Page Writer", func() {
		var signInPage *signInPageWriter

		BeforeEach(func() {
			errorTmpl, err := template.New("").Parse("{{.Title}}")
			Expect(err).ToNot(HaveOccurred())
			errorPage := &errorPageWriter{
				template: errorTmpl,
			}

			tmpl, err := template.New("").Parse("{{.ProxyPrefix}} {{.ProviderName}} {{.SignInMessage}} {{.Footer}} {{.Version}} {{.Redirect}} {{.CustomLogin}} {{.LogoData}}")
			Expect(err).ToNot(HaveOccurred())

			signInPage = &signInPageWriter{
				template:         tmpl,
				errorPageWriter:  errorPage,
				proxyPrefix:      "/prefix/",
				providerName:     "My Provider",
				signInMessage:    "Sign In Here",
				footer:           "Custom Footer Text",
				version:          "v0.0.0-test",
				displayLoginForm: true,
				logoData:         "Logo Data",
			}
		})

		Context("WriteSignInPage", func() {
			It("Writes the template to the response writer", func() {
				recorder := httptest.NewRecorder()
				signInPage.WriteSignInPage(recorder, "/redirect")

				body, err := ioutil.ReadAll(recorder.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(body)).To(Equal("/prefix/ My Provider Sign In Here Custom Footer Text v0.0.0-test /redirect true Logo Data"))
			})

			It("Writes an error if the template can't be rendered", func() {
				// Overwrite the template with something bad
				tmpl, err := template.New("").Parse("{{.Unknown}}")
				Expect(err).ToNot(HaveOccurred())
				signInPage.template = tmpl

				recorder := httptest.NewRecorder()
				signInPage.WriteSignInPage(recorder, "/redirect")

				body, err := ioutil.ReadAll(recorder.Result().Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(body)).To(Equal("Internal Server Error"))
			})
		})
	})

	Context("loadCustomLogo", func() {
		var customDir string

		const fakeImageData = "Fake Image Data"

		BeforeEach(func() {
			var err error
			customDir, err = ioutil.TempDir("", "oauth2-proxy-sign-in-page-test")
			Expect(err).ToNot(HaveOccurred())

			for _, ext := range []string{".svg", ".png", ".jpg", ".jpeg", ".gif"} {
				fileName := filepath.Join(customDir, fmt.Sprintf("logo%s", ext))
				Expect(ioutil.WriteFile(fileName, []byte(fakeImageData), 0600)).To(Succeed())
			}
		})

		AfterEach(func() {
			Expect(os.RemoveAll(customDir)).To(Succeed())
		})

		type loadCustomLogoTableInput struct {
			logoPath     string
			expectedErr  error
			expectedData string
		}

		DescribeTable("should load the logo based on configuration", func(in loadCustomLogoTableInput) {
			logoPath := in.logoPath
			if strings.HasPrefix(logoPath, "customDir/") {
				logoPath = filepath.Join(customDir, strings.TrimLeft(logoPath, "customDir/"))
			}

			data, err := loadCustomLogo(logoPath)
			if in.expectedErr != nil {
				Expect(err).To(MatchError(in.expectedErr.Error()))
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
			Expect(data).To(Equal(in.expectedData))
		},
			Entry("with no custom logo path", loadCustomLogoTableInput{
				logoPath:     "",
				expectedErr:  nil,
				expectedData: defaultLogoData,
			}),
			Entry("when disabling the logo display", loadCustomLogoTableInput{
				logoPath:     "-",
				expectedErr:  nil,
				expectedData: "",
			}),
			Entry("with an svg custom logo", loadCustomLogoTableInput{
				logoPath:     "customDir/logo.svg",
				expectedErr:  nil,
				expectedData: fakeImageData,
			}),
			Entry("with a png custom logo", loadCustomLogoTableInput{
				logoPath:     "customDir/logo.png",
				expectedErr:  nil,
				expectedData: "<img src=\"data:image/png;base64,RmFrZSBJbWFnZSBEYXRh\" alt=\"Logo\" />",
			}),
			Entry("with a jpg custom logo", loadCustomLogoTableInput{
				logoPath:     "customDir/logo.jpg",
				expectedErr:  nil,
				expectedData: "<img src=\"data:image/jpeg;base64,RmFrZSBJbWFnZSBEYXRh\" alt=\"Logo\" />",
			}),
			Entry("with a jpeg custom logo", loadCustomLogoTableInput{
				logoPath:     "customDir/logo.jpeg",
				expectedErr:  nil,
				expectedData: "<img src=\"data:image/jpeg;base64,RmFrZSBJbWFnZSBEYXRh\" alt=\"Logo\" />",
			}),
			Entry("with a gif custom logo", loadCustomLogoTableInput{
				logoPath:     "customDir/logo.gif",
				expectedErr:  errors.New("unknown extension: \".gif\", supported extensions are .svg, .jpg, .jpeg and .png"),
				expectedData: "",
			}),
			Entry("when the logo does not exist", loadCustomLogoTableInput{
				logoPath:     "unknown.svg",
				expectedErr:  errors.New("could not read logo file: open unknown.svg: no such file or directory"),
				expectedData: "",
			}),
		)
	})
})
