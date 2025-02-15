package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

// Define the FrontMatter struct
type FrontMatter struct {
	Title     string   `yaml:"title,omitempty"`
	Updated   string   `yaml:"updated,omitempty"`
	Created   string   `yaml:"created,omitempty"`
	Author    string   `yaml:"author,omitempty"`
	Latitude  float64  `yaml:"latitude,omitempty"`
	Longitude float64  `yaml:"longitude,omitempty"`
	Altitude  float64  `yaml:"altitude,omitempty"`
	Tags      []string `yaml:"tags,omitempty"`
	NoteTags  string   `yaml:"note-tags,omitempty"`
	Completed bool     `yaml:"completed?,omitempty"`
	// Add more fields as needed
}

// ProcessFrontMatter processes the front matter of a Markdown file
func ProcessFrontMatter(content string, cleanupFrontmatter bool) string {
	var frontMatter FrontMatter
	var filteredContent strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(content))
	inFrontMatter := false
	var frontMatterLines []string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			inFrontMatter = !inFrontMatter
			if !inFrontMatter {
				// Parse the front matter
				frontMatterContent := strings.Join(frontMatterLines, "\n")
				if err := yaml.Unmarshal([]byte(frontMatterContent), &frontMatter); err != nil {
					log.Printf("Failed to parse front matter: %v", err)
					return ""
				}

				// Logseq-ify the frontmatter tags for easier use in Logseq
				// frontMatter.Tags prefix tags with #
				for i, tag := range frontMatter.Tags {
					frontMatter.Tags[i] = "#" + tag
				}
				// convert frontMatter.Tags to string
				stringifiedTags := strings.Join(frontMatter.Tags, " ")
				// add stringifiedTags to frontMatter
				frontMatter.NoteTags = stringifiedTags

				// Remove specified fields if the cleanupFrontmatter flag is set
				if cleanupFrontmatter {
					frontMatter.Latitude = 0
					frontMatter.Longitude = 0
					frontMatter.Altitude = 0
					frontMatter.Title = ""
					frontMatter.Author = ""
					frontMatter.Tags = nil
				}

				// Marshal the front matter back to YAML
				updatedFrontMatter, err := yaml.Marshal(&frontMatter)
				if err != nil {
					log.Printf("Failed to marshal front matter: %v", err)
					return ""
				}

				// Write the updated front matter
				filteredContent.WriteString("---\n")
				filteredContent.Write(updatedFrontMatter)
				filteredContent.WriteString("---\n")
			}
			continue
		}

		if inFrontMatter {
			frontMatterLines = append(frontMatterLines, line)
		} else {
			filteredContent.WriteString(line + "\n")
			// Define the regex pattern to extract the filename from the Markdown link - this should work most of the time :see_no_evil:
			var re = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
			// Check for references to files in the assets folder
			if strings.Contains(line, "../assets/") {
				// Extract the asset path using regex
				matches := re.FindStringSubmatch(line)
				if len(matches) > 1 {
					assetPath := matches[2]
					assetFullPath := filepath.Join("joplin-input", "assets", assetPath)
					// Replace "assets" with "_resources" in assetFullPath - in order to check if they are existing in the source
					assetFullPath = strings.Replace(assetFullPath, "assets", "_resources", -1)
					log.Printf("Asset path: %s", assetFullPath)
					if _, err := os.Stat(assetFullPath); err == nil {
						// Copy the asset file to the output directory
						outputAssetPath := filepath.Join("logseq-output", "assets", assetPath)
						if err := os.MkdirAll(filepath.Dir(outputAssetPath), 0755); err != nil {
							log.Printf("Failed to create directory %s: %v", filepath.Dir(outputAssetPath), err)
							return ""
						}
						input, err := os.ReadFile(assetFullPath)
						if err != nil {
							log.Printf("Failed to read asset file %s: %v", assetFullPath, err)
							return ""
						}
						if err := os.WriteFile(outputAssetPath, input, 0644); err != nil {
							log.Printf("Failed to write asset file %s: %v", outputAssetPath, err)
							return ""
						}
					} else {
						log.Printf("Asset file %s does not exist", assetFullPath)
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Failed to scan file content: %v", err)
		return ""
	}
	return filteredContent.String()
}

func main() {
	// Dedfine source and destination directories
	sourceDirectory := "joplin-input"
	destinationDirectory := "logseq-output"

	// Define replacements
	keywordReplacements := map[string]string{
		"_resources": "assets", // Replace _resources with assets - because logseq uses the asset folder
	}

	// Parse command-line arguments
	cleanupFrontmatter := flag.Bool("frontmatter-cleanup", false, "Cleanup Frontmatter")
	flag.Parse()

	// Create the subdirectories if they don't exist
	if err := os.MkdirAll(sourceDirectory, 0755); err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.MkdirAll(destinationDirectory, 0755); err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}
	// create assets directory in destinationDirectory
	if err := os.MkdirAll(filepath.Join(destinationDirectory, "assets"), 0755); err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}
	// Walk through the subdirectory
	err := filepath.Walk(sourceDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the _resources directory
		if info.IsDir() && info.Name() == "_resources" {
			return filepath.SkipDir
		}

		// Check if the file has a .md extension
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			// Read the file content
			content, err := ioutil.ReadFile(path)
			if err != nil {
				log.Printf("Failed to read file %s: %v", path, err)
				return nil
			}

			// Perform keyword replacement
			updatedContent := string(content)
			for keyword, replacement := range keywordReplacements {
				updatedContent = strings.ReplaceAll(updatedContent, keyword, replacement)
			}

			// Process the front matter
			updatedContent = ProcessFrontMatter(updatedContent, *cleanupFrontmatter)

			// Construct the output file path
			relPath, err := filepath.Rel(sourceDirectory, path)
			if err != nil {
				log.Printf("Failed to get relative path for %s: %v", path, err)
				return nil
			}

			// Prefix the output file with the folder name of the input subdirectory
			// the three ___ are used in logseq to implement a folder structure
			dirName := filepath.Dir(relPath)
			baseName := filepath.Base(relPath)
			outputFileName := strings.ReplaceAll(dirName, string(os.PathSeparator), "___") + "___" + baseName
			outputFilePath := filepath.Join(destinationDirectory, "pages", outputFileName)
			log.Printf("Output file path: %s", outputFilePath)

			// Create the output directory if it doesn't exist
			if err := os.MkdirAll(filepath.Dir(outputFilePath), 0755); err != nil {
				log.Printf("Failed to create directory %s: %v", filepath.Dir(outputFilePath), err)
				return nil
			}

			// Write the updated content to the output directory
			if err := ioutil.WriteFile(outputFilePath, []byte(updatedContent), 0644); err != nil {
				log.Printf("Failed to write file %s: %v", outputFilePath, err)
				return nil
			}
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Failed to walk through directory: %v", err)
	}

	fmt.Println("Logseqification for all Markdown files done.")
}
