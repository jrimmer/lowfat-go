package docker

import (
	"testing"

	"github.com/jrimmer/lowfat-go/lf"
)

func TestPsUltraColumnExtract(t *testing.T) {
	// header + one row with a multi-token PORTS column and trailing NAME.
	in := "CONTAINER ID   IMAGE   COMMAND   CREATED   STATUS   PORTS   NAMES\n" +
		"abc123 img \"cmd\" 2 days ago Up 2 days 0.0.0.0:5432->5432/tcp, [::]:5432->5432/tcp web-1\n"
	s := shell{}
	out, err := s.RunShell("printf 'NAME STATUS\\n'\ntail -n +2 | awk '{print $NF, $(NF-2)}'", in, &lf.ExecCtx{Level: lf.Ultra})
	if err != nil {
		t.Fatal(err)
	}
	// $NF = web-1 (name), $(NF-2) = third-from-last = the first PORTS token.
	want := "NAME STATUS\nweb-1 0.0.0.0:5432->5432/tcp,"
	if out != want {
		t.Errorf("got %q\nwant %q", out, want)
	}
}

func TestImagesUltraColumnExtract(t *testing.T) {
	in := "REPOSITORY TAG IMAGE ID CREATED SIZE\n" +
		"alpine latest 93efcf633c54 2 days ago 14MB\n"
	s := shell{}
	out, err := s.RunShell("printf 'REPO TAG SIZE\\n'\ntail -n +2 | awk '{print $1, $2, $(NF-1)}'", in, &lf.ExecCtx{Level: lf.Ultra})
	if err != nil {
		t.Fatal(err)
	}
	// $1=alpine $2=latest $(NF-1)=second-from-last="ago"? No: fields are
	// alpine latest 93efcf633c54 2 days ago 14MB -> NF=7, $(NF-1)=$6="ago".
	want := "REPO TAG SIZE\nalpine latest ago"
	if out != want {
		t.Errorf("got %q\nwant %q", out, want)
	}
}

func TestCollapseSpaces(t *testing.T) {
	s := shell{}
	out, err := s.RunShell("sed 's/  */ /g'", "a    b   c\nx  y\n", &lf.ExecCtx{Level: lf.Full})
	if err != nil {
		t.Fatal(err)
	}
	if out != "a b c\nx y" {
		t.Errorf("got %q", out)
	}
}

func TestRelField(t *testing.T) {
	f := []string{"a", "b", "c", "d"} // NF=4
	if relField(f, 0) != "d" {        // $NF
		t.Errorf("$NF: got %q want d", relField(f, 0))
	}
	if relField(f, -1) != "c" { // $(NF-1)
		t.Errorf("$(NF-1): got %q want c", relField(f, -1))
	}
	if relField(f, -2) != "b" { // $(NF-2)
		t.Errorf("$(NF-2): got %q want b", relField(f, -2))
	}
	if relField(f, -9) != "" {
		t.Errorf("out of range should be empty")
	}
}
