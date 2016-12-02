package main

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/gruntwork-io/docs/file"
	"github.com/gruntwork-io/docs/errors"
	"github.com/gruntwork-io/docs/logger"
	"github.com/gruntwork-io/docs/globs"
	"github.com/gruntwork-io/docs/nav"
	"github.com/gruntwork-io/docs/gruntwork_package"
)

// TODO: Copy _content files into tmp _input folder

func ProcessFiles(opts *Opts) error {
	var err error

	packages, err := getGruntworkPackagesSlice(opts.RepoManifestPath)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	rootNavFolder := nav.NewRootFolder()

	// Walk all files, copy non-markdown files ("files") and load all markdown files ("pages") into a nav tree
	err = filepath.Walk(opts.InputPath, func(fullInputPath string, info os.FileInfo, fileErr error) error {
		relInputPath, err := file.GetPathRelativeTo(fullInputPath, opts.InputPath)
		if err != nil {
			return err
		} else if shouldSkipPath(relInputPath, opts) {
			logger.Logger.Printf("Skipping path %s\n", relInputPath)
			return nil
		} else {
			file := nav.NewFile(relInputPath, fullInputPath)
			err := file.PopulateOutputPath()
			if err != nil {
				// TODO: Neither the Type Assertion nor the error return works as expected here. Error:
				// runtime: goroutine stack exceeds 1000000000-byte limit
				// fatal error: stack overflow
				// ...
				// github.com/gruntwork-io/docs/nav.FileInputPathDidNotMatchAnyRegEx.Error(0x32e6bd, 0x1, 0xc440200420, 0x6185a)
				//	/Users/josh/go/src/github.com/gruntwork-io/docs/nav/file.go:126 +0x6a fp=0xc4402003d8 sp=0xc440200370
				// github.com/gruntwork-io/docs/nav.(*FileInputPathDidNotMatchAnyRegEx).Error(0xc42ada6370, 0x357138, 0xc42ada5d40)
				// <autogenerated>:6 +0x5b fp=0xc440200418 sp=0xc4402003d8

				//if noMatchErr, ok := err.(nav.FileInputPathDidNotMatchAnyRegEx); ok {
				//	logger.Logger.Printf("WARNING: File %s did not match any RegEx while processing.\nFull Error: %s\n", fullInputPath, noMatchErr)
				//} else {
				//	return err
				//}
			}

			if file.IsFile() {
				if err = file.WriteToOutputPath(opts.InputPath, opts.OutputPath); err != nil {
					return errors.WithStackTrace(err)
				}
			}

			if file.IsPage() {
				page := file.GetAsPage(rootNavFolder)
				if err = page.PopulateProperties(); err != nil {
					return errors.WithStackTrace(err)
				}

				if err = page.AddToNavTree(); err != nil {
					return errors.WithStackTrace(err)
				}
			}

			return nil
		}
	})
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// Now that our nav tree is constructed, populate the page bodies
	err = rootNavFolder.PopulateChildrenPageBodyProperties(opts.OutputPath, packages)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// Generate HTML from the NavTree files
	err = rootNavFolder.WriteChildrenHtmlToOutputhPath(rootNavFolder, opts.OutputPath)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	// Copy HTML assets into the output directory
	err = file.CopyFiles(opts.HtmlPath + "/css", opts.OutputPath + "/_assets/css")
	if err != nil {
		return errors.WithStackTrace(err)
	}

	err = file.CopyFiles(opts.HtmlPath + "/img", opts.OutputPath + "/_assets/img")
	if err != nil {
		return errors.WithStackTrace(err)
	}

	err = file.CopyFiles(opts.HtmlPath + "/favicons", opts.OutputPath + "/")
	if err != nil {
		return errors.WithStackTrace(err)
	}

	return nil
}

// Return true if this is a file or folder we should skip completely in the processing step.
func shouldSkipPath(path string, opts *Opts) bool {
	return path == opts.InputPath || globs.MatchesGlobs(path, opts.Excludes)
}

// Given a filepath, return a slice of GruntworkPackages
func getGruntworkPackagesSlice(packagesFilePath string) ([]gruntwork_package.GruntworkPackage, error) {
	var packages []gruntwork_package.GruntworkPackage

	jsonString, err := file.ReadFile(packagesFilePath)
	if err != nil {
		return packages, errors.WithStackTrace(err)
	}

	packages, err = gruntwork_package.GetSliceFromJson(jsonString)
	if err != nil {
		return packages, errors.WithStackTrace(err)
	}

	return packages, nil
}



//// This function will walk all the files specified in opt.InputPath and load them into the desired NavTree
//func LoadDocsIntoNavTree(opts *Opts) error {
//	return filepath.Walk(opts.InputPath, func(path string, info os.FileInfo, err error) error {
//		relPath, err := file.GetPathRelativeTo(path, opts.InputPath)
//		if err != nil {
//			return err
//		} else if shouldSkipPath(relPath, opts) {
//			fmt.Printf("Skipping path %s\n", relPath)
//			return nil
//		} else {
//			allDocFileTypes := docfile.CreateAllDocFileTypes(path, relPath)
//
//			for _, docFile := range allDocFileTypes {
//				if docFile.IsMatch() {
//
//
//					if err = docFile.Copy(opts.OutputPath); err != nil {
//						return errors.WithStackTrace(err)
//					}
//					return nil
//				}
//			}
//
//			// No DocFile could be created from the given relPath
//			logger.Logger.Printf("Ignoring %s", relPath)
//			return nil
//		}
//	})
//}

//
//// Check whether the given path matches the given RegEx. We panic if there's an error (versus returning a bool and an
//// error) to keep the if-else statement in ProcessDocs simpler.
//func checkRegex(path string, regexStr string) bool {
//	regex := regexp.MustCompile(regexStr)
//	return regex.MatchString(path)
//}
//
//// Return the output path for a GlobalDoc file. See TestGetGlobalDocOutputPath for expected output.
//func getGlobalDocOutputPath(path string) (string, error) {
//	var outputPath string
//
//	regex := regexp.MustCompile(IS_GLOBAL_DOC_REGEX)
//	submatches := regex.FindAllStringSubmatch(path, -1)
//
//	if len(submatches) != 1 || len(submatches[0]) != 2 {
//		return outputPath, WithStackTrace(RegExReturnedUnexpectedNumberOfMatches(IS_GLOBAL_DOC_REGEX))
//	}
//
//	outputPath = submatches[0][1]
//
//	return outputPath, nil
//}
//
//// Return the output path for a ModuleDoc file. See TestGetModuleDocExampleOutputPath for expected output.
//func getModuleDocOutputPath(path string) (string, error) {
//	var outputPath string
//
//	regex := regexp.MustCompile(IS_MODULE_DOC_REGEX)
//	submatches := regex.FindAllStringSubmatch(path, -1)
//
//	if len(submatches) != 1 || len(submatches[0]) != 3 {
//		return outputPath, errors.New("Module documents must exist in the path /packages/<package-name>/modules/<module-name>/_docs/. Any subfolders in /_docs will generate an error.")
//	}
//
//	// Full string: packages/module-vpc/modules/vpc-app/module-doc.md
//	// This part: packages/module-vpc/modules/vpc-app
//	modulePath := submatches[0][1]
//	modulePath = strings.Replace(modulePath, "modules/", "", 1)
//
//	// Full string: packages/module-vpc/modules/vpc-app/module-doc.md
//	// This part: module-doc.md
//	fileName := submatches[0][2]
//
//	return modulePath + "/" + fileName, nil
//}
//

// // Generate the documentation output for the given file into opts.OutputPath. If file is a documentation file, this will
// // copy the file largely unchanged, other than some placeholder text prepended and some URL tweaks. If file is a
// // non-documentation file, its contents will be replaced completely by placeholder text.
// func generateDocsForFile(file string, info os.FileInfo, opts *Opts) error {
// 	var contents []byte
// 	var err error
// 	var outPath = path.Join(opts.OutputPath, file)

// 	if MatchesGlobs(file, opts.DocPatterns) {
// 		Logger.Printf("Copying documentation file %s to %s without changes, except module-XXX URLs will be replaced with module-XXX-public.", file, outPath)
// 		contents, err = getContentsForDocumentationFile(file, opts)
// 	} else {
// 		Logger.Printf("Copying non-documentation file %s to %s and replacing its contents with placeholder text.", file, outPath)
// 		contents, err = getContentsForNonDocumentationFile(file, opts)
// 	}

// 	if err != nil {
// 		return err
// 	} else {
// 		return writeFileWithSamePermissions(outPath, contents, info)
// 	}
// }

// // Write the given contents to the given file path with the permissions in the given FileInfo
// func writeFileWithSamePermissions(file string, contents []byte, info os.FileInfo) error {
// 	return WithStackTrace(ioutil.WriteFile(file, contents, info.Mode()))
// }

// // Get the contents for a documentation file. These contents should be unchanged from the original, other than:
// //
// // 1. We prepend some placeholder text explaining where this file comes from
// // 2. In Markdown files, we replace any URLs to private module-XXX repos with URLs to the equivalent module-XXX-public
// //    repos
// func getContentsForDocumentationFile(file string, opts *Opts) ([]byte, error) {
// 	fullPath := path.Join(opts.InputPath, file)

// 	bytes, err := ioutil.ReadFile(fullPath)
// 	if err != nil {
// 		return []byte{}, WithStackTrace(err)
// 	}

// 	isText, err := IsTextFile(fullPath)
// 	if err != nil {
// 		return []byte{}, WithStackTrace(err)
// 	}

// 	// Return binaries, such as images, unchanged
// 	if !isText {
// 		return bytes, nil
// 	}

// 	// contents := string(bytes)
// 	// if path.Ext(file) == ".md" {
// 	// 	contents = ReplacePrivateGitHubUrlsWithPublicUrlsInMarkdown(contents)
// 	// }

// 	// text := CreatePlaceholderTextForFile(file, opts)
// 	// if len(contents) > 0 {
// 	// 	text = text + "\n\n" + contents
// 	// }

// 	//return []byte(text), nil
// 	return bytes, nil
// }

// // Get the contents for a non-documentation file. We replace the contents of source files entirely with placeholder
// // text.
// func getContentsForNonDocumentationFile(file string, opts *Opts) ([]byte, error) {
// 	return []byte(CreatePlaceholderTextForFile(file, opts)), nil
// }

// custom error types

type RegExReturnedUnexpectedNumberOfMatches string
func (regex RegExReturnedUnexpectedNumberOfMatches) Error() string {
	return fmt.Sprintf("The Regular Expression \"%s\" returned a different number of matches than we expected.\n", string(regex))
}