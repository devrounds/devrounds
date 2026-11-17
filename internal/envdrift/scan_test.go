package envdrift

import "testing"

func checkLines(t *testing.T, found map[string][]Loc, want map[string]int) {
	t.Helper()
	for name, line := range want {
		locs, ok := found[name]
		if !ok {
			t.Errorf("%s not found", name)
			continue
		}
		if locs[0].Line != line {
			t.Errorf("%s at line %d, want %d", name, locs[0].Line, line)
		}
	}
}

func TestScanFileNode(t *testing.T) {
	src := "const a = process.env.DATABASE_URL;\n" +
		"const b = process.env[\"JWT_SECRET\"];\n" +
		"const {\n  REDIS_URL,\n  PORT = 3000,\n} = process.env;\n" +
		"const home = process.env.HOME;\n"
	found := map[string][]Loc{}
	scanFile(found, "app.js", src)
	checkLines(t, found, map[string]int{"DATABASE_URL": 1, "JWT_SECRET": 2, "REDIS_URL": 3, "PORT": 3})
	if _, ok := found["HOME"]; ok {
		t.Error("HOME should be ignored")
	}
}

func TestScanFilePython(t *testing.T) {
	src := "import os\n" +
		"url = os.environ[\"DB_URL\"]\n" +
		"key = os.environ.get('API_KEY', 'x')\n" +
		"mode = os.getenv(\"APP_MODE\")\n"
	found := map[string][]Loc{}
	scanFile(found, "app.py", src)
	checkLines(t, found, map[string]int{"DB_URL": 2, "API_KEY": 3, "APP_MODE": 4})
}

func TestScanFileSkipsDynamicNames(t *testing.T) {
	src := "const v = process.env[name];\n" +
		"const { X } = process.env.FOO;\n" +
		"x = os.getenv(name)\n"
	found := map[string][]Loc{}
	scanFile(found, "app.js", src)
	if _, ok := found["X"]; ok {
		t.Error("X must not be read from process.env.FOO")
	}
	if _, ok := found["FOO"]; !ok {
		t.Error("FOO should be found")
	}
	if len(found) != 1 {
		t.Errorf("found %d names, want 1: %v", len(found), found)
	}
}
