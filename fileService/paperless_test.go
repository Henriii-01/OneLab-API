package fileService

import "testing"

func TestFindPaperlessContainerNameFromDockerPsOutput(t *testing.T) {
	output := "paperless-webserver\tabc123\npaperless-db\tdef456\n"

	got, err := findPaperlessContainerNameFromDockerPsOutput(output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "paperless-webserver" {
		t.Fatalf("expected paperless-webserver, got %q", got)
	}
}

func TestFindPaperlessContainerNameFromDockerPsOutput_UsesFallbackPaperlessName(t *testing.T) {
	output := "some-other\tzzz\npaperless\tqqq\n"

	got, err := findPaperlessContainerNameFromDockerPsOutput(output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "paperless" {
		t.Fatalf("expected paperless, got %q", got)
	}
}
