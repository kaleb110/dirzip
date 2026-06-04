package main

import (
	"flag"
	"fmt"
	"os"
)

// multiFlag lets --exclude be specified multiple times on the command line
// (or once with comma-separated values).
type multiFlag []string

func (m *multiFlag) String() string {
	if m == nil || len(*m) == 0 {
		return ""
	}
	s := ""
	for i, v := range *m {
		if i > 0 {
			s += ","
		}
		s += v
	}
	return s
}

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func main() {
	var (
		output   = flag.String("output", "", "destination zip file path (required)")
		password = flag.String("password", "", "encrypt the archive with this password (zip: requires alexmullins/zip build tag)")
		excludes multiFlag
	)

	flag.Var(&excludes, "exclude", "directory name to exclude (may be repeated, or comma-separated)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: dirzip [flags] <source-directory>\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  dirzip -output out.zip ./myproject\n")
		fmt.Fprintf(os.Stderr, "  dirzip -output out.zip -exclude node_modules -exclude .git ./myproject\n")
		fmt.Fprintf(os.Stderr, "  dirzip -output out.zip -exclude node_modules,.git ./myproject\n")
		fmt.Fprintf(os.Stderr, "  dirzip -output out.zip -password s3cr3t ./myproject\n")
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	sourceDir := flag.Arg(0)

	if *output == "" {
		fmt.Fprintln(os.Stderr, "error: -output flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Verify source directory exists.
	info, err := os.Stat(sourceDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot access source directory: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "error: %s is not a directory\n", sourceDir)
		os.Exit(1)
	}

	opts := ArchiveOptions{
		SourceDir:   sourceDir,
		OutputPath:  *output,
		ExcludeDirs: ParseExcludes(excludes),
		Password:    *password,
	}

	archiver := selectArchiver(*password)

	if *password != "" && !archiver.SupportsEncryption() {
		fmt.Fprintln(os.Stderr, "error: selected format does not support encryption")
		os.Exit(1)
	}

	fmt.Printf("Archiving %s → %s\n", sourceDir, *output)
	if len(opts.ExcludeDirs) > 0 {
		fmt.Printf("Excluding: %v\n", excludes)
	}
	if *password != "" {
		fmt.Println("Encryption: enabled")
	}

	if err := archiver.Archive(opts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Done. Archive written to %s\n", *output)
}

// selectArchiver chooses the appropriate Archiver based on the requested
// features. Extend this function to support additional formats (tar.gz, 7z…).
func selectArchiver(password string) Archiver {
	// Currently only ZIP is implemented.
	// If a password is requested and you have the encrypted fork, swap to
	// EncryptedZipArchiver here:
	//   if password != "" { return EncryptedZipArchiver{} }
	_ = password
	return ZipArchiver{}
}
