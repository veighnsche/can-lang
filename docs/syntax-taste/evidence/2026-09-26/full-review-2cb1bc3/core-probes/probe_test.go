package emit
import (
 "os"
 "path/filepath"
 "testing"
 "github.com/veighnsche/can-lang/compiler/internal/project"
 "github.com/veighnsche/can-lang/compiler/internal/check"
)
func TestFreshCoreProbe(t *testing.T) {
 root := os.Getenv("CAN_REVIEW_PROJECT")
 graph, err := project.Load(root); if err != nil { t.Fatal(err) }
 program, err := check.CheckProgram(graph); if err != nil { t.Fatal(err) }
 for _, mode := range []string{"prod", "assert"} {
   artifacts, err := ProgramModules(program,"runtime",liveRuntimeArtifacts(t))
   if mode == "assert" { artifacts, err = AssertionModules(program,"runtime",liveRuntimeArtifacts(t)) }
   if err != nil { t.Fatal(err) }
   for _, artifact := range artifacts {
     p := filepath.Join(root,mode,artifact.Path)
     if err := os.MkdirAll(filepath.Dir(p),0700); err != nil {t.Fatal(err)}
     if err := os.WriteFile(p,artifact.Bytes,0600); err != nil {t.Fatal(err)}
   }
 }
 t.Logf("checked and emitted %s",root)
}
