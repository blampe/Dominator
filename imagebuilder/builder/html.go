package builder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	"github.com/Symantec/Dominator/lib/format"
)

func (b *Builder) writeHtml(writer io.Writer) {
	fmt.Fprintf(writer,
		"Number of image streams: <a href=\"showImageStreams\">%d</a><p>\n",
		b.getNumNormalStreams())
	currentBuilds := make([]string, 0)
	goodBuilds := make(map[string]buildResultType)
	failedBuilds := make(map[string]buildResultType)
	b.buildResultsLock.RLock()
	for name := range b.currentBuildLogs {
		currentBuilds = append(currentBuilds, name)
	}
	for name, result := range b.lastBuildResults {
		if result.error == nil {
			goodBuilds[name] = result
		} else {
			failedBuilds[name] = result
		}
	}
	b.buildResultsLock.RUnlock()
	if len(currentBuilds) > 0 {
		fmt.Fprintln(writer, "Current image builds:<br>")
		fmt.Fprintln(writer, `<table border="1">`)
		fmt.Fprintln(writer, "  <tr>")
		fmt.Fprintln(writer, "    <th>Image Stream</th>")
		fmt.Fprintln(writer, "    <th>Build log</th>")
		fmt.Fprintln(writer, "  </tr>")
		for _, streamName := range currentBuilds {
			fmt.Fprintf(writer, "  <tr>\n")
			fmt.Fprintf(writer, "    <td>%s</td>\n", streamName)
			fmt.Fprintf(writer,
				"    <td><a href=\"showCurrentBuildLog?%s#bottom\">log</a></td>\n",
				streamName)
			fmt.Fprintf(writer, "  </tr>\n")
		}
		fmt.Fprintln(writer, "</table><br>")
	}
	if len(failedBuilds) > 0 {
		streamNames := make([]string, 0, len(failedBuilds))
		for streamName := range failedBuilds {
			streamNames = append(streamNames, streamName)
		}
		sort.Strings(streamNames)
		fmt.Fprintln(writer, "Failed image builds:<br>")
		fmt.Fprintln(writer, `<table border="1">`)
		fmt.Fprintln(writer, "  <tr>")
		fmt.Fprintln(writer, "    <th>Image Stream</th>")
		fmt.Fprintln(writer, "    <th>Error</th>")
		fmt.Fprintln(writer, "    <th>Build log</th>")
		fmt.Fprintln(writer, "  </tr>")
		for _, streamName := range streamNames {
			result := failedBuilds[streamName]
			fmt.Fprintf(writer, "  <tr>\n")
			fmt.Fprintf(writer, "    <td>%s</td>\n", streamName)
			fmt.Fprintf(writer, "    <td>%s</td>\n", result.error)
			fmt.Fprintf(writer,
				"    <td><a href=\"showLastBuildLog?%s\">log</a></td>\n",
				streamName)
			fmt.Fprintf(writer, "  </tr>\n")
		}
		fmt.Fprintln(writer, "</table><br>")
	}
	if len(goodBuilds) > 0 {
		streamNames := make([]string, 0, len(goodBuilds))
		for streamName := range goodBuilds {
			streamNames = append(streamNames, streamName)
		}
		sort.Strings(streamNames)
		fmt.Fprintln(writer, "Successful image builds:<br>")
		fmt.Fprintln(writer, `<table border="1">`)
		fmt.Fprintln(writer, "  <tr>")
		fmt.Fprintln(writer, "    <th>Image Stream</th>")
		fmt.Fprintln(writer, "    <th>Name</th>")
		fmt.Fprintln(writer, "    <th>Build log</th>")
		fmt.Fprintln(writer, "  </tr>")
		for _, streamName := range streamNames {
			result := goodBuilds[streamName]
			fmt.Fprintf(writer, "  <tr>\n")
			fmt.Fprintf(writer, "    <td>%s</td>\n", streamName)
			fmt.Fprintf(writer,
				"    <td><a href=\"http://%s/showImage?%s\">%s</a></td>\n",
				b.imageServerAddress, result.imageName, result.imageName)
			fmt.Fprintf(writer,
				"    <td><a href=\"showLastBuildLog?%s\">log</a></td>\n",
				streamName)
			fmt.Fprintf(writer, "  </tr>\n")
		}
		fmt.Fprintln(writer, "</table><br>")
	}
}

func (b *Builder) showImageStream(writer io.Writer, streamName string) {
	stream := b.getNormalStream(streamName)
	if stream == nil {
		fmt.Fprintf(writer, "<b>Stream: %s does not exist!</b>\n", streamName)
		return
	}
	fmt.Fprintf(writer, "<h3>Information for stream: %s</h3>\n", streamName)
	fmt.Fprintf(writer, "Manifest URL: <code>%s</code><br>\n",
		stream.ManifestUrl)
	fmt.Fprintf(writer, "Manifest Directory: <code>%s</code><br>\n",
		stream.ManifestDirectory)
	startTime := time.Now()
	manifestRoot, manifestDirectory, err := stream.getManifest(b, streamName,
		"", new(bytes.Buffer))
	if err != nil {
		fmt.Fprintf(writer, "<b>%s</b><br>\n", err)
		return
	}
	defer os.RemoveAll(manifestRoot)
	loadTime := time.Since(startTime)
	manifestDirectory = path.Join(manifestRoot, manifestDirectory)
	manifestFilename := path.Join(manifestDirectory, "manifest")
	manifestBytes, err := ioutil.ReadFile(manifestFilename)
	if err != nil {
		fmt.Fprintf(writer, "<b>%s</b><br>\n", err)
		return
	}
	var manifest manifestType
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		fmt.Fprintf(writer, "<b>%s</b><br>\n", err)
		return
	}
	sourceStream := b.getNormalStream(manifest.SourceImage)
	if sourceStream == nil {
		fmt.Fprintf(writer, "SourceImage: <code>%s</code><br>\n",
			manifest.SourceImage)
	} else {
		fmt.Fprintf(writer,
			"SourceImage: <a href=\"showImageStream?%s\"><code>%s</code></a><br>\n",
			manifest.SourceImage, manifest.SourceImage)
	}
	fmt.Fprintln(writer, "Contents of <code>manifest</code> file:<br>")
	fmt.Fprintln(writer, "<pre>")
	writer.Write(manifestBytes)
	fmt.Fprintln(writer, "</pre><p class=\"clear\">")
	packagesFile, err := os.Open(path.Join(manifestDirectory, "package-list"))
	if err != nil {
		fmt.Fprintf(writer, "<b>%s</b><br>\n", err)
		return
	}
	defer packagesFile.Close()
	fmt.Fprintln(writer, "Contents of <code>package-list</code> file:<br>")
	fmt.Fprintln(writer, "<pre>")
	io.Copy(writer, packagesFile)
	fmt.Fprintln(writer, "</pre><p class=\"clear\">")
	if size, err := getTreeSize(manifestRoot); err != nil {
		fmt.Fprintf(writer, "<b>%s</b><br>\n", err)
		return
	} else {
		speed := float64(size) / loadTime.Seconds()
		fmt.Fprintf(writer, "Repository size: %s, took: %s (%s/s)<br>\n",
			format.FormatBytes(size), format.Duration(loadTime),
			format.FormatBytes(uint64(speed)))
	}
	if size, err := getTreeSize(manifestDirectory); err != nil {
		fmt.Fprintf(writer, "<b>%s</b><br>\n", err)
		return
	} else {
		fmt.Fprintf(writer, "Manifest tree size: %s<br>\n",
			format.FormatBytes(size))
	}
}

func (b *Builder) showImageStreams(writer io.Writer) {
	streamNames := b.listNormalStreamNames()
	sort.Strings(streamNames)
	fmt.Fprintln(writer, `<table border="1">`)
	fmt.Fprintln(writer, "  <tr>")
	fmt.Fprintln(writer, "    <th>Image Stream</th>")
	fmt.Fprintln(writer, "    <th>ManifestUrl</th>")
	fmt.Fprintln(writer, "    <th>ManifestDirectory</th>")
	fmt.Fprintln(writer, "  </tr>")
	for _, streamName := range streamNames {
		imageStream := b.getNormalStream(streamName)
		if imageStream == nil {
			continue
		}
		fmt.Fprintf(writer, "  <tr>\n")
		fmt.Fprintf(writer,
			"    <td><a href=\"showImageStream?%s\">%s</a></td>\n",
			streamName, streamName)
		fmt.Fprintf(writer, "    <td>%s</td>\n", imageStream.ManifestUrl)
		fmt.Fprintf(writer, "    <td>%s</td>\n", imageStream.ManifestDirectory)
		fmt.Fprintf(writer, "  </tr>\n")
	}
	fmt.Fprintln(writer, "</table><br>")
}

func getTreeSize(dirname string) (uint64, error) {
	var size uint64
	err := filepath.Walk(dirname,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			size += uint64(info.Size())
			return nil
		})
	if err != nil {
		return 0, err
	}
	return size, nil
}
