//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	versionRe = regexp.MustCompile(`-[0-9]{1,3}-g[0-9a-f]{5,10}`)
	goarch    string
	goos      string
	version   string = "v1"
	// deb & rpm does not support semver so have to handle their version a little differently
	linuxPackageVersion   string = "v1"
	linuxPackageIteration string = ""
	race                  bool   = false
	workingDir            string
	serverBinaryName      string = "oracle_collector"
)

const minGoVersion = 1.11

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)

	ensureGoPath()
	// readVersionFromPackageJson()
	// readVersionFromChangelog()
	readVersionFromGit()
	log.Printf("Version: %s, Linux Version: %s, Package Iteration: %s\n", version, linuxPackageVersion, linuxPackageIteration)

	flag.StringVar(&goarch, "goarch", runtime.GOARCH, "GOARCH")
	flag.StringVar(&goos, "goos", runtime.GOOS, "GOOS")
	flag.BoolVar(&race, "race", race, "Use race detector")
	flag.Parse()

	if flag.NArg() == 0 {
		log.Println("Usage: go run build.go build")
		return
	}

	workingDir, _ = os.Getwd()

	for _, cmd := range flag.Args() {
		switch cmd {
		case "setup":
			setup()

		case "build":
			pkg := "./pkg/"
			clean()
			build(pkg, []string{}, []string{})

		case "build-static":
			pkg := "./pkg/"
			clean()
			build(pkg, []string{}, []string{"-linkmode", "external", "-extldflags", "-static"})
			//"-linkmode external -extldflags -static"
		case "test":
			test("./pkg/...")

		case "package":
			os.Mkdir("./dist", 0o755)
			createLinuxPackages()
			sha1FilesInDist()
		case "pkg-min-tar":
			os.Mkdir("./dist", 0o755)
			createMinTar()
			sha1FilesInDist()
		case "pkg-rpm":
			os.Mkdir("./dist", 0o755)
			createRpmPackages()
			sha1FilesInDist()
		case "pkg-deb":
			os.Mkdir("./dist", 0o755)
			createDebPackages()
			sha1FilesInDist()
		case "sha1-dist":
			sha1FilesInDist()
		case "latest":
			os.Mkdir("./dist", 0o755)
			createLinuxPackages()
			makeLatestDistCopies()
			sha1FilesInDist()

		case "clean":
			clean()

		default:
			log.Fatalf("Unknown command %q", cmd)
		}
	}
}

func makeLatestDistCopies() {
	rpmIteration := "-1"
	if linuxPackageIteration != "" {
		rpmIteration = "-" + linuxPackageIteration
	}

	runError("cp", "dist/oracle_collector_"+version+"_amd64.deb", "dist/oracle_collector_latest_amd64.deb")
	runError("cp", "dist/oracle_collector-"+linuxPackageVersion+rpmIteration+".x86_64.rpm", "dist/oracle_collector-latest-1.x86_64.rpm")
	runError("cp", "dist/oracle_collector-"+version+".linux-x64.tar.gz", "dist/oracle_collector-latest.linux-x64.tar.gz")
}

func readVersionFromPackageJson() {
	reader, err := os.Open("package.json")
	if err != nil {
		log.Fatal("Failed to open package.json")
		return
	}
	defer reader.Close()

	jsonObj := map[string]interface{}{}
	jsonParser := json.NewDecoder(reader)

	if err := jsonParser.Decode(&jsonObj); err != nil {
		log.Fatal("Failed to decode package.json")
	}

	version = jsonObj["version"].(string)
	linuxPackageVersion = version
	linuxPackageIteration = ""

	// handle pre version stuff (deb / rpm does not support semver)
	parts := strings.Split(version, "-")

	if len(parts) > 1 {
		linuxPackageVersion = parts[0]
		linuxPackageIteration = parts[1]
	}
}

func readVersionFromChangelog() {
	cmd := "grep ' *v *[0-9]*\\.[0-9]*\\.[0-9]' CHANGELOG.md | sed 's/# v *\\([0-9]*\\.[0-9]*\\.[0-9]\\) .*/\\1/g'| head -1"
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		log.Fatal(err)
	}

	linuxPackageVersion = strings.TrimSpace(string(out))
	version = linuxPackageVersion
	linuxPackageIteration = ""

	// handle pre version stuff (deb / rpm does not support semver)
	parts := strings.Split(version, "-")

	if len(parts) > 1 {
		linuxPackageVersion = parts[0]
		linuxPackageIteration = parts[1]
	}
}

func readVersionFromGit() {
	cmd := "git describe --abbrev=0 --tag"
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		log.Fatal(err)
	}

	linuxPackageVersion = strings.TrimSpace(string(out))
	version = linuxPackageVersion
	linuxPackageIteration = ""

	// handle pre version stuff (deb / rpm does not support semver)
	parts := strings.Split(version, "-")

	if len(parts) > 1 {
		linuxPackageVersion = parts[0]
		linuxPackageIteration = parts[1]
	}
}

type linuxPackageOptions struct {
	packageType            string
	homeDir                string
	binPath                string
	configDir              string
	configFilePath         string
	etcDefaultPath         string
	etcDefaultFilePath     string
	initdScriptFilePath    string
	systemdServiceFilePath string

	postinstSrc    string
	initdScriptSrc string
	defaultFileSrc string
	systemdFileSrc string

	depends []string
}

func createDebPackages() {
	createFpmPackage(linuxPackageOptions{
		packageType:            "deb",
		homeDir:                "/usr/share/oracle_collector",
		binPath:                "/usr/sbin/oracle_collector",
		configDir:              "/etc/oracle_collector",
		configFilePath:         "/etc/oracle_collector/oracle_collector.toml",
		etcDefaultPath:         "/etc/default",
		etcDefaultFilePath:     "/etc/default/oracle_collector",
		initdScriptFilePath:    "/etc/init.d/oracle_collector",
		systemdServiceFilePath: "/usr/lib/systemd/system/oracle_collector.service",

		postinstSrc:    "packaging/deb/control/postinst",
		initdScriptSrc: "packaging/deb/init.d/oracle_collector",
		defaultFileSrc: "packaging/deb/default/oracle_collector",
		systemdFileSrc: "packaging/deb/systemd/oracle_collector.service",

		depends: []string{"adduser"},
	})
}

func createRpmPackages() {
	createFpmPackage(linuxPackageOptions{
		packageType:            "rpm",
		homeDir:                "/usr/share/oracle_collector",
		binPath:                "/usr/sbin/oracle_collector",
		configDir:              "/etc/oracle_collector",
		configFilePath:         "/etc/oracle_collector/oracle_collector.toml",
		etcDefaultPath:         "/etc/sysconfig",
		etcDefaultFilePath:     "/etc/sysconfig/oracle_collector",
		initdScriptFilePath:    "/etc/init.d/oracle_collector",
		systemdServiceFilePath: "/usr/lib/systemd/system/oracle_collector.service",

		postinstSrc:    "packaging/rpm/control/postinst",
		initdScriptSrc: "packaging/rpm/init.d/oracle_collector",
		defaultFileSrc: "packaging/rpm/sysconfig/oracle_collector",
		systemdFileSrc: "packaging/rpm/systemd/oracle_collector.service",

		depends: []string{"initscripts"},
	})
}

func createLinuxPackages() {
	createDebPackages()
	createRpmPackages()
}

func createMinTar() {
	packageRoot, _ := ioutil.TempDir("", "oracle_collector-linux-pack")
	// create directories
	runPrint("mkdir", "-p", filepath.Join(packageRoot, "/opt/oracle_collector"))
	runPrint("mkdir", "-p", filepath.Join(packageRoot, "/opt/oracle_collector/conf"))
	runPrint("mkdir", "-p", filepath.Join(packageRoot, "/opt/oracle_collector/bin"))
	runPrint("mkdir", "-p", filepath.Join(packageRoot, "/opt/oracle_collector/log"))
	// runPrint("mkdir", "-p", filepath.Join(packageRoot, "/opt/oracle_collector/public"))
	runPrint("cp", "conf/sample.oracle_collector.toml", filepath.Join(packageRoot, "/opt/oracle_collector/conf"))
	runPrint("cp", "bin/oracle_collector", filepath.Join(packageRoot, "/opt/oracle_collector/bin"))
	runPrint("cp", "bin/oracle_collector.md5", filepath.Join(packageRoot, "/opt/oracle_collector/bin"))
	// runPrint("cp", "-a", filepath.Join(workingDir, "public")+"/.", filepath.Join(packageRoot, "/opt/oracle_collector/public"))
	tarname := fmt.Sprintf("dist/oracle_collector-%s-%s_%s_%s.tar.gz", version, getGitSha(), runtime.GOOS, runtime.GOARCH)
	runPrint("tar", "zcvf", tarname, "-C", packageRoot, ".")
	runPrint("rm", "-rf", packageRoot)
}

func createFpmPackage(options linuxPackageOptions) {
	packageRoot, _ := ioutil.TempDir("", "oracle_collector-linux-pack")

	// create directories
	runPrint("mkdir", "-p", filepath.Join(packageRoot, options.homeDir))
	runPrint("mkdir", "-p", filepath.Join(packageRoot, options.configDir))
	runPrint("mkdir", "-p", filepath.Join(packageRoot, "/etc/init.d"))
	runPrint("mkdir", "-p", filepath.Join(packageRoot, options.etcDefaultPath))
	runPrint("mkdir", "-p", filepath.Join(packageRoot, "/usr/lib/systemd/system"))
	runPrint("mkdir", "-p", filepath.Join(packageRoot, "/usr/sbin"))

	// copy binary
	runPrint("cp", "-p", filepath.Join(workingDir, "bin/"+serverBinaryName), filepath.Join(packageRoot, options.binPath))
	// copy init.d script
	runPrint("cp", "-p", options.initdScriptSrc, filepath.Join(packageRoot, options.initdScriptFilePath))
	// copy environment var file
	runPrint("cp", "-p", options.defaultFileSrc, filepath.Join(packageRoot, options.etcDefaultFilePath))
	// copy systemd filerunPrint("cp", "-a", filepath.Join(workingDir, "tmp")+"/.", filepath.Join(packageRoot, options.homeDir))
	runPrint("cp", "-p", options.systemdFileSrc, filepath.Join(packageRoot, options.systemdServiceFilePath))
	// copy release files
	// runPrint("cp", "-a", filepath.Join(workingDir+"/public"), filepath.Join(packageRoot, options.homeDir))
	// remove bin path
	runPrint("rm", "-rf", filepath.Join(packageRoot, options.homeDir, "bin"))
	// copy sample ini file to /etc/oracle_collector
	runPrint("cp", "conf/sample.oracle_collector.toml", filepath.Join(packageRoot, options.configFilePath))

	args := []string{
		"-s", "dir",
		"--description", "An InfluxDB data syncronization/Replication tool with HTTP admin API ",
		"-C", packageRoot,
		"--vendor", "oracle_collector",
		"--url", "http://github.org/toni-moreno/oracle_collector",
		"--license", "Apache 2.0",
		"--maintainer", "toni.moreno@gmail.com",
		"--config-files", options.configFilePath,
		"--config-files", options.initdScriptFilePath,
		"--config-files", options.etcDefaultFilePath,
		"--config-files", options.systemdServiceFilePath,
		"--after-install", options.postinstSrc,
		"--name", "oracle_collector",
		"--version", linuxPackageVersion,
		"-p", "./dist",
	}

	if linuxPackageIteration != "" {
		args = append(args, "--iteration", linuxPackageIteration)
	}

	// add dependenciesj
	for _, dep := range options.depends {
		args = append(args, "--depends", dep)
	}

	args = append(args, ".")

	fmt.Println("Creating package: ", options.packageType)
	runPrint("fpm", append([]string{"-t", options.packageType}, args...)...)
}

func verifyGitRepoIsClean() {
	rs, err := runError("git", "ls-files", "--modified")
	if err != nil {
		log.Fatalf("Failed to check if git tree was clean, %v, %v\n", string(rs), err)
		return
	}
	count := len(string(rs))
	if count > 0 {
		log.Fatalf("Git repository has modified files, aborting")
	}

	log.Println("Git repository is clean")
}

func ensureGoPath() {
	if os.Getenv("GOPATH") == "" {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		gopath := filepath.Clean(filepath.Join(cwd, "../../../../"))
		log.Println("GOPATH is", gopath)
		os.Setenv("GOPATH", gopath)
	}
}

func ChangeWorkingDir(dir string) {
	os.Chdir(dir)
}

func setup() {
	runPrint("go", "get", "-v", "github.com/tools/godep")
	// pending to check if these following 3 lines are really needed
	runPrint("go", "get", "-v", "github.com/blang/semver")
}

func test(pkg string) {
	setBuildEnv()
	runPrint("go", "test", "-short", "-timeout", "60s", pkg)
}

func build(pkg string, tags []string, flags []string) {
	binary := "./bin/" + serverBinaryName
	if goos == "windows" {
		binary += ".exe"
	}

	rmr(binary, binary+".md5")
	args := []string{"build", "-ldflags", ldflags(flags)}
	if len(tags) > 0 {
		args = append(args, "-tags", strings.Join(tags, ","))
	}
	if race {
		args = append(args, "-race")
	}
	args = append(args, "-v")
	args = append(args, "-o", binary)
	args = append(args, pkg)
	setBuildEnv()

	runPrint("go", "version")
	runPrint("go", args...)

	// Create an md5 checksum of the binary, to be included in the archive for
	// automatic upgrades.
	err := md5File(binary)
	if err != nil {
		log.Fatal(err)
	}
}

func ldflags(flags []string) string {
	var b bytes.Buffer
	b.WriteString("-w")
	b.WriteString(fmt.Sprintf(" -X github.com/toni-moreno/oracle_collector/pkg/agent.Version=%s", version))
	b.WriteString(fmt.Sprintf(" -X github.com/toni-moreno/oracle_collector/pkg/agent.Commit=%s", getGitSha()))
	b.WriteString(fmt.Sprintf(" -X github.com/toni-moreno/oracle_collector/pkg/agent.BuildStamp=%d", buildStamp()))
	for _, f := range flags {
		b.WriteString(fmt.Sprintf(" %s", f))
	}
	return b.String()
}

func rmr(paths ...string) {
	for _, path := range paths {
		log.Println("rm -r", path)
		os.RemoveAll(path)
	}
}

func clean() {
	//	rmr("bin", "Godeps/_workspace/pkg", "Godeps/_workspace/bin")
	// rmr("public")
	// rmr("tmp")
	rmr(filepath.Join(os.Getenv("GOPATH"), fmt.Sprintf("pkg/%s_%s/github.com/toni-moreno/oracle_collector", goos, goarch)))
}

func setBuildEnv() {
	os.Setenv("GOOS", goos)
	if strings.HasPrefix(goarch, "armv") {
		os.Setenv("GOARCH", "arm")
		os.Setenv("GOARM", goarch[4:])
	} else {
		os.Setenv("GOARCH", goarch)
	}
	if goarch == "386" {
		os.Setenv("GO386", "387")
	}
	/*wd, err := os.Getwd()
	if err != nil {
		log.Println("Warning: can't determine current dir:", err)
		log.Println("Build might not work as expected")
	}
	os.Setenv("GOPATH", fmt.Sprintf("%s%c%s", filepath.Join(wd, "Godeps", "_workspace"), os.PathListSeparator, os.Getenv("GOPATH")))*/
	log.Println("GOPATH=" + os.Getenv("GOPATH"))
}

func getGitSha() string {
	v, err := runError("git", "rev-parse", "--short", "HEAD")
	if err != nil {
		return "unknown-dev"
	}
	v = versionRe.ReplaceAllFunc(v, func(s []byte) []byte {
		s[0] = '+'
		return s
	})
	return string(v)
}

func buildStamp() int64 {
	bs, err := runError("git", "show", "-s", "--format=%ct")
	if err != nil {
		return time.Now().Unix()
	}
	s, _ := strconv.ParseInt(string(bs), 10, 64)
	return s
}

func buildArch() string {
	os := goos
	if os == "darwin" {
		os = "macosx"
	}
	return fmt.Sprintf("%s-%s", os, goarch)
}

func run(cmd string, args ...string) []byte {
	bs, err := runError(cmd, args...)
	if err != nil {
		log.Println(cmd, strings.Join(args, " "))
		log.Println(string(bs))
		log.Fatal(err)
	}
	return bytes.TrimSpace(bs)
}

func runError(cmd string, args ...string) ([]byte, error) {
	ecmd := exec.Command(cmd, args...)
	bs, err := ecmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return bytes.TrimSpace(bs), nil
}

func runPrint(cmd string, args ...string) {
	log.Println(cmd, strings.Join(args, " "))
	ecmd := exec.Command(cmd, args...)
	ecmd.Stdout = os.Stdout
	ecmd.Stderr = os.Stderr
	err := ecmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func md5File(file string) error {
	fd, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fd.Close()

	h := md5.New()
	_, err = io.Copy(h, fd)
	if err != nil {
		return err
	}

	out, err := os.Create(file + ".md5")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(out, "%x\n", h.Sum(nil))
	if err != nil {
		return err
	}

	return out.Close()
}

func sha1FilesInDist() {
	filepath.Walk("./dist", func(path string, f os.FileInfo, err error) error {
		if strings.Contains(path, ".sha1") == false {
			sha1File(path)
		}
		return nil
	})
}

func sha1File(file string) error {
	fd, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fd.Close()

	h := sha1.New()
	_, err = io.Copy(h, fd)
	if err != nil {
		return err
	}

	out, err := os.Create(file + ".sha1")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(out, "%x\n", h.Sum(nil))
	if err != nil {
		return err
	}

	return out.Close()
}
