package tools

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/ldez/go-git-cmd-wrapper/v2/clone"
	"github.com/ldez/go-git-cmd-wrapper/v2/git"
)

const (
	OAS_REPO_NAME = "stackit-api-specifications"
	OAS_REPO      = "https://github.com/stackitcloud/stackit-api-specifications.git"
	GEN_REPO_NAME = "stackit-sdk-generator"
	GEN_REPO      = "https://github.com/stackitcloud/stackit-sdk-generator.git"
)

type version struct {
	verString string
	major     int
	minor     int
}

func Build() error {
	slog.Info("Starting Builder")
	root, err := getRoot()
	if err != nil {
		log.Fatal(err)
	}
	if root == nil || *root == "" {
		return fmt.Errorf("unable to determine root directory from git")
	}
	slog.Info("Using root directory", "dir", *root)

	slog.Info("Cleaning up old generator directory")
	err = os.RemoveAll(path.Join(*root, GEN_REPO_NAME))
	if err != nil {
		return err
	}

	slog.Info("Cleaning up old packages directory")
	err = os.RemoveAll(path.Join(*root, "pkg"))
	if err != nil {
		return err
	}

	slog.Info("Creating generator dir", "dir", fmt.Sprintf("%s/%s", *root, GEN_REPO_NAME))
	genDir, err := createGeneratorDir(*root, GEN_REPO, GEN_REPO_NAME)
	if err != nil {
		return err
	}

	slog.Info("Creating oas dir", "dir", fmt.Sprintf("%s/%s", *root, OAS_REPO_NAME))
	repoDir, err := createRepoDir(genDir, OAS_REPO, OAS_REPO_NAME)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	slog.Info("Retrieving versions from subdirs")
	// TODO - major
	verMap, err := getVersions(repoDir)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	slog.Info("Reducing to only latest or highest")
	res, err := getOnlyLatest(verMap)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	slog.Info("Creating OAS dir")
	err = os.MkdirAll(path.Join(genDir, "oas"), 0755)
	if err != nil {
		return err
	}

	slog.Info("Copying OAS files")
	for service, item := range res {
		baseService := strings.TrimSuffix(service, "alpha")
		baseService = strings.TrimSuffix(baseService, "beta")
		itemVersion := fmt.Sprintf("v%d%s", item.major, item.verString)
		if item.minor != 0 {
			itemVersion = itemVersion + "" + strconv.Itoa(item.minor)
		}
		srcFile := path.Join(
			repoDir,
			"services",
			baseService,
			itemVersion,
			fmt.Sprintf("%s.json", baseService),
		)
		dstFile := path.Join(genDir, "oas", fmt.Sprintf("%s.json", service))
		_, err = copyFile(srcFile, dstFile)
		if err != nil {
			return fmt.Errorf(err.Error())
		}
	}

	slog.Info("Cleaning up", "dir", repoDir)
	err = os.RemoveAll(filepath.Dir(repoDir))
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	slog.Info("Changing dir", "dir", genDir)
	err = os.Chdir(genDir)
	if err != nil {
		return err
	}

	slog.Info("Calling make", "command", "generate-go-sdk")
	cmd := exec.Command("make", "generate-go-sdk")
	var stdOut, stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr

	if err = cmd.Start(); err != nil {
		slog.Error("cmd.Start", err)
		return err
	}

	if err = cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			slog.Error("cmd.Wait", "code", exitErr.ExitCode())
			return fmt.Errorf(stdErr.String())
		} else {
			slog.Error("cmd.Wait", "err", err)
			return err
		}
	}

	slog.Info("Cleaning up go.mod and go.sum files")
	cleanDir := path.Join(genDir, "sdk-repo-updated", "services")
	dirEntries, err := os.ReadDir(cleanDir)
	if err != nil {
		return err
	}
	for _, entry := range dirEntries {
		if entry.IsDir() {
			err = deleteFiles(
				path.Join(cleanDir, entry.Name(), "go.mod"),
				path.Join(cleanDir, entry.Name(), "go.sum"),
			)
			if err != nil {
				return err
			}
		}
	}

	slog.Info("Changing dir", "dir", *root)
	err = os.Chdir(*root)
	if err != nil {
		return err
	}

	slog.Info("Rearranging package directories")
	err = os.MkdirAll(path.Join(*root, "pkg"), 0755)
	if err != nil {
		return err
	}
	srcDir := path.Join(genDir, "sdk-repo-updated", "services")
	items, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.IsDir() {
			slog.Info(" -> package", "name", item.Name())
			tgtDir := path.Join(*root, "pkg", item.Name())
			// no backup needed as we generate new
			//bakName := fmt.Sprintf("%s.%s", item.Name(), time.Now().Format("20060102-150405"))
			//if _, err = os.Stat(tgtDir); !os.IsNotExist(err) {
			//	err = os.Rename(
			//		tgtDir,
			//		path.Join(*root, "pkg", bakName),
			//	)
			//	if err != nil {
			//		return err
			//	}
			//}
			err = os.Rename(path.Join(srcDir, item.Name()), tgtDir)
			if err != nil {
				return err
			}

			// wait is placed outside now
			//if _, err = os.Stat(path.Join(*root, "pkg", bakName, "wait")); !os.IsNotExist(err) {
			//	slog.Info("    Copying wait subfolder")
			//	err = os.Rename(path.Join(*root, "pkg", bakName, "wait"), path.Join(tgtDir, "wait"))
			//	if err != nil {
			//		return err
			//	}
			//}
		}
	}

	slog.Info("Checking needed commands available")
	err = checkCommands([]string{"tfplugingen-framework", "tfplugingen-openapi"})
	if err != nil {
		return err
	}

	slog.Info("Generating service boilerplate")
	err = generateServiceFiles(*root, path.Join(*root, GEN_REPO_NAME))
	if err != nil {
		return err
	}

	slog.Info("Copying all service files")
	err = CopyDirectory(
		path.Join(*root, "generated", "internal", "services"),
		path.Join(*root, "stackit", "internal", "services"),
	)
	if err != nil {
		return err
	}

	err = createBoilerplate(*root, path.Join(*root, "stackit", "internal", "services"))
	if err != nil {
		return err
	}

	slog.Info("Finally removing temporary files and directories")
	//err = os.RemoveAll(path.Join(*root, "generated"))
	//if err != nil {
	//	slog.Error("RemoveAll", "dir", path.Join(*root, "generated"), "err", err)
	//	return err
	//}

	err = os.RemoveAll(path.Join(*root, GEN_REPO_NAME))
	if err != nil {
		slog.Error("RemoveAll", "dir", path.Join(*root, GEN_REPO_NAME), "err", err)
		return err
	}

	slog.Info("Done")
	return nil
}

type templateData struct {
	PackageName string
	NameCamel   string
	NamePascal  string
	NameSnake   string
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		panic(err)
	}
	return true
}

func createBoilerplate(rootFolder, folder string) error {
	services, err := os.ReadDir(folder)
	if err != nil {
		return err
	}
	for _, svc := range services {
		if !svc.IsDir() {
			continue
		}
		resources, err := os.ReadDir(path.Join(folder, svc.Name()))
		if err != nil {
			return err
		}

		handleDS := false
		handleRes := false
		foundDS := false
		foundRes := false
		for _, res := range resources {
			if !res.IsDir() {
				continue
			}

			resourceName := res.Name()

			dsFile := path.Join(folder, svc.Name(), res.Name(), "datasources_gen", fmt.Sprintf("%s_data_source_gen.go", res.Name()))
			handleDS = fileExists(dsFile)

			resFile := path.Join(folder, svc.Name(), res.Name(), "resources_gen", fmt.Sprintf("%s_resource_gen.go", res.Name()))
			handleRes = fileExists(resFile)

			dsGoFile := path.Join(folder, svc.Name(), res.Name(), "datasource.go")
			foundDS = fileExists(dsGoFile)

			resGoFile := path.Join(folder, svc.Name(), res.Name(), "resource.go")
			foundRes = fileExists(resGoFile)

			if handleDS && !foundDS {
				slog.Info("Creating missing datasource.go", "service", svc.Name(), "resource", resourceName)
				if !ValidateSnakeCase(resourceName) {
					return errors.New("resource name is invalid")
				}

				tplName := "data_source_scaffold.gotmpl"
				err = writeTemplateToFile(
					tplName,
					path.Join(rootFolder, "tools", "templates", tplName),
					path.Join(folder, svc.Name(), res.Name(), "datasource.go"),
					&templateData{
						PackageName: svc.Name(),
						NameCamel:   ToCamelCase(resourceName),
						NamePascal:  ToPascalCase(resourceName),
						NameSnake:   resourceName,
					},
				)
				if err != nil {
					panic(err)
				}
			}

			if handleRes && !foundRes {
				slog.Info("Creating missing resource.go", "service", svc.Name(), "resource", resourceName)
				if !ValidateSnakeCase(resourceName) {
					return errors.New("resource name is invalid")
				}

				tplName := "resource_scaffold.gotmpl"
				err = writeTemplateToFile(
					tplName,
					path.Join(rootFolder, "tools", "templates", tplName),
					path.Join(folder, svc.Name(), res.Name(), "resource.go"),
					&templateData{
						PackageName: svc.Name(),
						NameCamel:   ToCamelCase(resourceName),
						NamePascal:  ToPascalCase(resourceName),
						NameSnake:   resourceName,
					},
				)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func ucfirst(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func writeTemplateToFile(tplName, tplFile, outFile string, data *templateData) error {
	fn := template.FuncMap{
		"ucfirst": ucfirst,
	}

	tmpl, err := template.New(tplName).Funcs(fn).ParseFiles(tplFile)
	if err != nil {
		return err
	}

	var f *os.File
	f, err = os.Create(outFile)
	if err != nil {
		return err
	}

	err = tmpl.Execute(f, *data)
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}
	return nil
}

func generateServiceFiles(rootDir, generatorDir string) error {
	// slog.Info("Generating specs folder")
	err := os.MkdirAll(path.Join(rootDir, "generated", "specs"), 0755)
	if err != nil {
		return err
	}

	specs, err := os.ReadDir(path.Join(rootDir, "service_specs"))
	if err != nil {
		return err
	}
	for _, spec := range specs {
		if spec.IsDir() {
			continue
		}
		// slog.Info("Checking spec", "name", spec.Name())
		r := regexp.MustCompile(`^([a-z-]+)_(.*)_config.yml$`)
		matches := r.FindAllStringSubmatch(spec.Name(), -1)
		if matches != nil {
			fileName := matches[0][0]
			service := matches[0][1]
			resource := matches[0][2]
			slog.Info(
				"Found service spec",
				"name",
				spec.Name(),
				"service",
				service,
				"resource",
				resource,
			)

			for _, part := range []string{"alpha", "beta"} {
				oasFile := path.Join(generatorDir, "oas", fmt.Sprintf("%s%s.json", service, part))
				if _, err = os.Stat(oasFile); !os.IsNotExist(err) {
					// slog.Info("found matching oas", "service", service, "version", part)
					scName := fmt.Sprintf("%s%s", service, part)
					scName = strings.ReplaceAll(scName, "-", "")
					err = os.MkdirAll(path.Join(rootDir, "generated", "internal", "services", scName, resource), 0755)
					if err != nil {
						return err
					}

					// slog.Info("Generating openapi spec json")
					specFile := path.Join(rootDir, "generated", "specs", fmt.Sprintf("%s_%s_spec.json", scName, resource))
					cmd := exec.Command(
						"tfplugingen-openapi",
						"generate",
						"--config",
						path.Join(rootDir, "service_specs", fileName),
						"--output",
						specFile,
						oasFile,
					)
					out, err := cmd.Output()
					if err != nil {
						fmt.Printf("%s\n", string(out))
						return err
					}

					// slog.Info("Creating terraform service resource files folder")
					tgtFolder := path.Join(rootDir, "generated", "internal", "services", scName, resource, "resources_gen")
					err = os.MkdirAll(tgtFolder, 0755)
					if err != nil {
						return err
					}

					// slog.Info("Generating terraform service resource files")
					cmd2 := exec.Command(
						"tfplugingen-framework",
						"generate",
						"resources",
						"--input",
						specFile,
						"--output",
						tgtFolder,
						"--package",
						scName,
					)
					var stdOut, stdErr bytes.Buffer
					cmd2.Stdout = &stdOut
					cmd2.Stderr = &stdErr

					if err = cmd2.Start(); err != nil {
						slog.Error("cmd.Start", err)
						return err
					}

					if err = cmd2.Wait(); err != nil {
						var exitErr *exec.ExitError
						if errors.As(err, &exitErr) {
							slog.Error("cmd.Wait", "code", exitErr.ExitCode())
							return fmt.Errorf(stdErr.String())
						} else {
							slog.Error("cmd.Wait", "err", err)
							return err
						}
					}

					// slog.Info("Creating terraform service datasource files folder")
					tgtFolder = path.Join(rootDir, "generated", "internal", "services", scName, resource, "datasources_gen")
					err = os.MkdirAll(tgtFolder, 0755)
					if err != nil {
						return err
					}

					// slog.Info("Generating terraform service resource files")
					cmd3 := exec.Command(
						"tfplugingen-framework",
						"generate",
						"data-sources",
						"--input",
						specFile,
						"--output",
						tgtFolder,
						"--package",
						scName,
					)
					var stdOut3, stdErr3 bytes.Buffer
					cmd3.Stdout = &stdOut3
					cmd3.Stderr = &stdErr3

					if err = cmd3.Start(); err != nil {
						slog.Error("cmd.Start", err)
						return err
					}

					if err = cmd3.Wait(); err != nil {
						var exitErr *exec.ExitError
						if errors.As(err, &exitErr) {
							slog.Error("cmd.Wait", "code", exitErr.ExitCode())
							return fmt.Errorf(stdErr.String())
						} else {
							slog.Error("cmd.Wait", "err", err)
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func checkCommands(commands []string) error {
	for _, commandName := range commands {
		if commandExists(commandName) {
			slog.Info("found", "command", commandName)
		} else {
			return fmt.Errorf("missing command %s\n", commandName)
		}
	}
	return nil
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func deleteFiles(fNames ...string) error {
	for _, fName := range fNames {
		if _, err := os.Stat(fName); !os.IsNotExist(err) {
			err = os.Remove(fName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func getOnlyLatest(m map[string]version) (map[string]version, error) {
	tmpMap := make(map[string]version)
	for k, v := range m {
		item, ok := tmpMap[k]
		if !ok {
			tmpMap[k] = v
		} else {
			if item.major == v.major && item.minor < v.minor {
				tmpMap[k] = v
			}
		}
	}
	return tmpMap, nil
}

func getVersions(dir string) (map[string]version, error) {
	res := make(map[string]version)
	children, err := os.ReadDir(path.Join(dir, "services"))
	if err != nil {
		return nil, err
	}

	for _, entry := range children {
		if entry.IsDir() {
			versions, err := os.ReadDir(path.Join(dir, "services", entry.Name()))
			if err != nil {
				return nil, err
			}
			m, err2 := extractVersions(entry.Name(), versions)
			if err2 != nil {
				return m, err2
			}
			for k, v := range m {
				res[k] = v
			}
		}
	}
	return res, nil
}

func extractVersions(service string, versionDirs []os.DirEntry) (map[string]version, error) {
	res := make(map[string]version)
	for _, vDir := range versionDirs {
		if vDir.IsDir() {
			r := regexp.MustCompile(`v([0-9]+)([a-z]+)([0-9]*)`)
			matches := r.FindAllStringSubmatch(vDir.Name(), -1)
			if matches == nil {
				continue
			}
			svc, ver, err := handleVersion(service, matches[0])
			if err != nil {
				return nil, err
			}

			if svc != nil && ver != nil {
				res[*svc] = *ver
			}
		}
	}
	return res, nil
}

func handleVersion(service string, match []string) (*string, *version, error) {
	if match == nil {
		fmt.Println("no matches")
		return nil, nil, nil
	}
	verString := match[2]
	if verString != "alpha" && verString != "beta" {
		return nil, nil, errors.New("unsupported version")
	}
	majVer, err := strconv.Atoi(match[1])
	if err != nil {
		return nil, nil, err
	}
	if match[3] == "" {
		match[3] = "0"
	}
	minVer, err := strconv.Atoi(match[3])
	if err != nil {
		return nil, nil, err
	}
	resStr := fmt.Sprintf("%s%s", service, verString)
	return &resStr, &version{verString: verString, major: majVer, minor: minVer}, nil
}

func createRepoDir(root, repoUrl, repoName string) (string, error) {
	oasTmpDir, err := os.MkdirTemp(root, "oas-tmp")
	if err != nil {
		return "", err
	}
	targetDir := path.Join(oasTmpDir, repoName)
	_, err = git.Clone(
		clone.Repository(repoUrl),
		clone.Directory(targetDir),
	)
	if err != nil {
		return "", err
	}
	return targetDir, nil
}

func createGeneratorDir(root, repoUrl, repoName string) (string, error) {
	targetDir := path.Join(root, repoName)
	_, err := git.Clone(
		clone.Repository(repoUrl),
		clone.Directory(targetDir),
	)
	if err != nil {
		return "", err
	}
	return targetDir, nil
}

func getRoot() (*string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	return &lines[0], nil
}
