package nav

import (
	"strings"
	"github.com/gruntwork-io/docs/util"
	"path/filepath"
	"fmt"
	"github.com/gruntwork-io/terragrunt/errors"
	"html/template"
	"regexp"
	"sort"
	"github.com/gruntwork-io/docs/gruntwork_package"
)

type Folder struct {
	OutputPath      string    // the path where this folder will exist when finally output
	Name            string    // the name of the folder
	ChildPages      []*Page   // the list of pages contained in this folder
	ChildFolders    []*Folder // the list of folders containers in this folder
	ParentFolder    *Folder   // the folder in which this folder resides
	IsRoot          bool      // true if this is the topmost folder
	IsPackageFolder bool      // true if this folder contains a Gruntwork Package
	IsModuleFolder  bool      // true if this folder contains a Gruntwork Module within a Gruntwork Package
}

// Add a childFolder to f
func (f *Folder) AddFolder(childFolder *Folder) {
	f.ChildFolders = append(f.ChildFolders, childFolder)
	childFolder.ParentFolder = f
}

// Add a childPage to f
func (f *Folder) AddPage(childPage *Page) {
	f.ChildPages = append(f.ChildPages, childPage)
	childPage.ParentFolder = f
}

// Add f to the given parentFolder
func (f *Folder) AddToFolder(parentFolder *Folder) {
	f.ParentFolder = parentFolder
	parentFolder.ChildFolders = append(parentFolder.ChildFolders, f)
}

// Returns true if this folder or any of its recursive children have the given folderName
func (f *Folder) ContainsFolderRecursive(folderName string) bool {
	if f.Name == folderName {
		return true
	}

	for _, folder := range f.ChildFolders {
		if folder.ContainsFolderRecursive(folderName) {
			return true
		}
	}

	return false
}

// Returns true if this folder or any of its direct children (but no further descendants) have the given folderName
func (f *Folder) HasChildFolder(folderName string) bool {
	for _, folder := range f.ChildFolders {
		if folder.Name == folderName {
			return true
		}
	}

	return false
}

// Returns the direct child folder of the given name, or nil if not found
func (f *Folder) GetChildFolder(folderName string) *Folder {
	for _, folder := range f.ChildFolders {
		if folder.Name == folderName {
			return folder
		}
	}

	return nil
}

// Returns the given folder if it exists in the current folder or any recursive child folder. Otherwise returns nil.
func (f *Folder) GetFolder(folderName string) *Folder {
	if f.Name == folderName {
		return f
	}

	for _, folder := range f.ChildFolders {
		if folder.GetFolder(folderName) != nil {
			return folder
		}
	}

	return nil
}

// Given a folderPath x/y/z, create each such folder if it does not already exist
func (f *Folder) CreateFolderIfNotExist(folderPath string) *Folder {
	folderPath = getStandardizedPath(folderPath)
	folderNameToCreate, numRemainingFolders := getTopFolderNameInPath(folderPath)

	// Base case
	if numRemainingFolders == 0 {
		if f.HasChildFolder(folderNameToCreate) {
			return f.GetChildFolder(folderNameToCreate)
		} else {
			newChildFolderPath := filepath.Join(f.OutputPath, folderNameToCreate)
			childFolder := NewFolder(newChildFolderPath, folderNameToCreate)
			f.AddFolder(childFolder)

			return childFolder
		}
	}

	// Recursive Case
	var childFolder *Folder

	if f.HasChildFolder(folderNameToCreate) {
		childFolder = f.GetChildFolder(folderNameToCreate)
	} else {
		newChildFolderPath := filepath.Join(f.OutputPath, folderNameToCreate)
		childFolder = NewFolder(newChildFolderPath, folderNameToCreate)
		f.AddFolder(childFolder)
	}

	folderPathTail := getFolderPathTail(folderPath)

	return childFolder.CreateFolderIfNotExist(folderPathTail)
}

// Output this folder and all its descendants as HTML
func (f *Folder) WriteChildrenHtmlToOutputhPath(rootFolder *Folder, rootOutputPath string) error {
	for _, page := range f.ChildPages {
		err := page.WriteFullPageHtmlToOutputPath(rootFolder, rootOutputPath)
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}

	for _, folder := range f.ChildFolders {
		err := folder.WriteChildrenHtmlToOutputhPath(rootFolder, rootOutputPath)
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}

	return nil
}

// Populate all page body properties in this folder's child pages, and the recursive children of its child folders
func (f *Folder) PopulateChildrenPageBodyProperties(rootOutputPath string, packages []gruntwork_package.GruntworkPackage) error {
	var err error

	for _, page := range f.ChildPages {
		if err = page.PopulateBodyProperties(rootOutputPath, packages); err != nil {
			return errors.WithStackTrace(err)
		}
	}

	for _, folder := range f.ChildFolders {
		if err = folder.PopulateChildrenPageBodyProperties(rootOutputPath, packages); err != nil {
			return errors.WithStackTrace(err)
		}
	}

	return nil
}

// Print the entire tree of a given folder
func (f *Folder) PrintFolderTree() {
	f.printFolderTreeAux(0)
}

// Helper function for printing a complete tree
func (f *Folder) printFolderTreeAux(folderDepth int) {
	fmt.Printf("%s", strings.Repeat("- ", folderDepth))
	fmt.Printf("FOLDER: %s\n", f.Name)

	for _, folder := range f.ChildFolders {
		folder.printFolderTreeAux(folderDepth + 1)
	}

	for _, page := range f.ChildPages {
		fmt.Printf("%s", strings.Repeat("- ", folderDepth + 1))
		fmt.Printf("%s\n", page.Title)
	}
}

// Print a nicely formatted string of the folder
func (f *Folder) PrintFolder() {
	var parentFolderName string
	var childFolders string
	var childPages string

	if f.ParentFolder != nil {
		parentFolderName = f.ParentFolder.Name
	}

	if f.ChildFolders != nil {
		childFolderNames := []string{}
		for _, childFolder := range f.ChildFolders {
			childFolderNames = append(childFolderNames, childFolder.Name)
		}
		childFolders = fmt.Sprintf("%v", childFolderNames)
	}

	if f.ChildPages != nil {
		childPageNames := []string{}
		for _, childPage := range f.ChildPages {
			childPageNames = append(childPageNames, childPage.Title)
		}
		childPages = fmt.Sprintf("%v", childPageNames)
	}

	fmt.Printf("[ name=%s, path=%s, parentFolder=%s, childFolders=%s, childPages=%s ]\n",
		f.Name,
		f.OutputPath,
		parentFolderName,
		childFolders,
		childPages,
	)
}

// Get a template.HTML of this Folder's childFolders and childPages
func (f *Folder) GetAsNavTreeHtml(activePage *Page) template.HTML {
	return template.HTML(f.getAsNavTreeHtmlAux(activePage))
}

// A helper function for GetAsNavTreeHtml
func (f *Folder) getAsNavTreeHtmlAux(activePage *Page) string {
	var htmlOutput string

	if f.IsRoot {
		f.ChildFolders = reorderFoldersToMatchTopLevelFolderOrdering(f.ChildFolders)
		sortPages(f.ChildPages)
	} else {
		sortFolders(f.ChildFolders)
		sortPages(f.ChildPages)
	}

	// Hide all subfolders under a package folder
	if len(f.ChildFolders) > 0 && f.IsPackageFolder {
		htmlOutput += "<ul class='hidden'>"
	} else if len(f.ChildFolders) > 0  {
		htmlOutput += "<ul>"
	}

	for _, childFolder := range f.ChildFolders {
		childFolderName := childFolder.Name
		if ! childFolder.IsModuleFolder {
			childFolderName = convertDashesToSpacesAndCapitalize(childFolderName)
		}

		cssClasses := ""
		if f.IsRoot {
			cssClasses = " top_level_folder"
		}
		if childFolder.IsPackageFolder {
			cssClasses = " package_folder"
		}
		if childFolder.IsModuleFolder {
			cssClasses = " module_folder"
		}

		htmlOutput += fmt.Sprintf("<li class='folder%s'><a href='#'>%s</a>", cssClasses, childFolderName)

		// Hide all pages
		if len(childFolder.ChildPages) > 0 {
			htmlOutput += "<ul class='hidden'>"
		}

		for _, childPage := range childFolder.ChildPages {
			childPageTitle := convertDashesToSpacesAndCapitalize(childPage.Title)
			if childPage == activePage {
				htmlOutput += fmt.Sprintf("<li class='page'><a class='active' href='#'>%s</a></li>", childPageTitle)
			} else {
				htmlOutput += fmt.Sprintf("<li class='page'><a href='%s'>%s</a></li>", activePage.GetRelPathToPage(childPage), childPageTitle)
			}
		}

		if len(childFolder.ChildPages) > 0 {
			htmlOutput += "</ul>"
		}

		htmlOutput += childFolder.getAsNavTreeHtmlAux(activePage)

		htmlOutput += "</li>"
	}

	if len(f.ChildFolders) > 0 {
		htmlOutput += "</ul>"
	}

	return htmlOutput
}

// Given a folderPath such as /x/y/z or ./x/y/z, return the top folder name
func getTopFolderNameInPath(folderPath string) (string, int) {
	folderPath = getStandardizedPath(folderPath)

	folderNames := strings.Split(folderPath, "/")
	numRemainingFolders := len(folderNames) - 1

	return folderNames[0], numRemainingFolders
}

// Given a folderPath such as /x/y/z or ./x/y/z, return the top folder name
func getFolderPathTail(folderPath string) string {
	folderPath = getStandardizedPath(folderPath)

	folderNames := strings.Split(folderPath, "/")
	folderNamesTail := util.GetStrSliceTail(folderNames)
	folderNamesTailStr := strings.Join(folderNamesTail, "/")

	return folderNamesTailStr
}

// Convert a path of the form ./x/y/z, /x/y/z, or x/y/z to the form x/y/z
func getStandardizedPath(path string) string {
	if strings.HasPrefix(path, "/") {
		path = strings.Replace(path, "/", "", 1)
	}

	if strings.HasPrefix(path, "./") {
		path = strings.Replace(path, "./", "", 1)
	}

	return path
}

// Return a new root folder
func NewRootFolder() *Folder {
	return &Folder{
		Name: "ROOT-FOLDER",
		IsRoot: true,
	}
}

// Return a generic new folder
func NewFolder(path, name string) *Folder {
	folder := &Folder{
		OutputPath: path,
		Name: name,
	}

	regex := regexp.MustCompile(OUTPUT_PATH_IS_PACKAGE_FOLDER_REGEX)
	if regex.MatchString(path) {
		folder.IsPackageFolder = true
	}

	regex = regexp.MustCompile(OUTPUT_PATH_IS_MODULE_FOLDER_REGEX)
	if regex.MatchString(path) {
		folder.IsModuleFolder = true
	}

	return folder
}

// Implement the Golang sort.interface on []*Folders so that we can sort folders alphabetically
type Folders []*Folder

func (folders Folders) Len() int {
	return len(folders)
}

func (folders Folders) Less(i, j int) bool {
	return folders[i].Name < folders[j].Name
}

func (folders Folders) Swap(i, j int) {
	folders[i], folders[j] = folders[j], folders[i]
}

// Sort the given slice of Folders
func sortFolders(folders Folders) {
	sort.Sort(folders)
}

// Implement the Golang sort.interface on []*Pages so that we can sort pages alphabetically
type Pages []*Page

func (pages Pages) Len() int {
	return len(pages)
}

// Always put "Overview" at the front of the list. Otherwise, sort by page Title.
func (pages Pages) Less(i, j int) bool {
	if pages[i].Title == "Overview" {
		return true
	} else if pages[j].Title == "Overview" {
		return false
	} else {
		return pages[i].Title < pages[j].Title
	}
}

func (pages Pages) Swap(i, j int) {
	pages[i], pages[j] = pages[j], pages[i]
}

// Sort the given slice of Pages
func sortPages(pages Pages) {
	sort.Sort(pages)
}