package shell

import (
	"log"
	"os/exec"
)

func RunSimpleCmd() string {
	//p := shellwords.NewParser(Parsebacktick=true)

	out, err := exec.Command("date").Output()
	if err != nil {
		log.Fatal(err)
	}
	return string(out)
}
