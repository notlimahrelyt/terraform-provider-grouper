package provider

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories wires the in-process provider into the
// acceptance test framework.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"grouper": providerserver.NewProtocol6WithError(New("acctest")()),
}

// approvedTestStemPattern is a guardrail: the test stem must obviously be a test
// namespace. This prevents acceptance tests from ever creating or destroying
// objects in a production folder.
var approvedTestStemPattern = regexp.MustCompile(`(?i)test`)

// testAccPreCheck validates that acceptance tests are explicitly enabled and
// that all required, safe environment variables are set. It fails the test
// early (t.Fatal) rather than risk touching an unintended namespace.
func testAccPreCheck(t *testing.T) {
	t.Helper()

	required := []string{"GROUPER_BASE_URL", "GROUPER_USERNAME", "GROUPER_PASSWORD", "GROUPER_TEST_STEM"}
	for _, key := range required {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			t.Fatalf("%s must be set for acceptance tests", key)
		}
	}

	testStem := os.Getenv("GROUPER_TEST_STEM")
	if !approvedTestStemPattern.MatchString(testStem) {
		t.Fatalf("GROUPER_TEST_STEM (%q) does not look like a test namespace; it must match %q. "+
			"Refusing to run destructive tests against a non-test stem.", testStem, approvedTestStemPattern.String())
	}
}

// testStem returns the configured, validated acceptance-test parent stem.
func testStem() string { return strings.TrimSpace(os.Getenv("GROUPER_TEST_STEM")) }

// testChildStem returns a full child stem path under the test stem.
func testChildStem(leaf string) string {
	return fmt.Sprintf("%s:%s", testStem(), leaf)
}

// Subject-related test inputs. Membership acceptance tests need real,
// resolvable subjects; they skip when these are not provided.
func testSubjectID() string  { return strings.TrimSpace(os.Getenv("GROUPER_TEST_SUBJECT_ID")) }
func testSubjectID2() string { return strings.TrimSpace(os.Getenv("GROUPER_TEST_SUBJECT_ID_2")) }
func testSubjectSourceID() string {
	return strings.TrimSpace(os.Getenv("GROUPER_TEST_SUBJECT_SOURCE_ID"))
}

// requireTestSubject skips the test unless a primary test subject is configured.
func requireTestSubject(t *testing.T) {
	t.Helper()
	if testSubjectID() == "" {
		t.Skip("GROUPER_TEST_SUBJECT_ID not set; skipping membership acceptance test")
	}
}
