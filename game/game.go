package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/akamensky/argparse"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/go-github/github"

	"gopkg.in/yaml.v2"
)

//////// main ////////

func main() {
	// read stdin for file to parse
	filename := parseArgs()

	printBanner("Gitub Read API call: " + filename)
	fetchAndLogJSON()

	printBanner("AWS API Read call: " + filename)
	listObjects()

	printBanner("File i/o Read input file: " + filename)
	contents := readFile(filename)

	printBanner("AWS API Write call: " + filename)
	putObject(contents)

	// Read YAML field from input file
	service := unmarshlManifest(contents)
	// access a param in that object
	configFile := service.Name
	printBanner("Read YAML field from input file: " + configFile)
	// reads 2nd file to stdout
	readFile(configFile)
}

//////// Const ////////

const LINE_LENGTH int = 50

//////// Structs ////////

type Manifest struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	}
	Name string `yaml:"name"`
}

//////// Parse Args ////////

// Returns the filename from the arguments to parse
func parseArgs() string {
	parser := argparse.NewParser("print", "Prints provided string to stdout")
	s := parser.String("s", "string", &argparse.Options{Required: true, Help: "String to print 's'"})
	err := parser.Parse(os.Args)
	check(err)
	return *s
}

// fetch a file from Github, loads JSON, and converts to a struct
func fetchAndLogJSON() {
	client := github.NewClient(nil)

	content, _, _, err := client.Repositories.GetContents(
		context.Background(), "angular", "angular-phonecat", "package.json", nil,
	)
	check(err)

	// decode content
	decoded, err := base64.StdEncoding.DecodeString(*content.Content)
	check(err)

	// convert JSON to struct
	jsPackage := JsPackage{}
	json.Unmarshal([]byte(decoded), &jsPackage)

	// log some attrs
	fmt.Println(jsPackage.Name)
	fmt.Println(jsPackage.Version)
}

type JsPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// S3 API call
func listObjects() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-1"),
	})
	check(err)

	// S3 service
	svc := s3.New(sess)

	// input args to API call
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String("d2a-config"),
		MaxKeys: aws.Int64(2),
	}

	// make API call
	result, err := svc.ListObjectsV2(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				fmt.Println(s3.ErrCodeNoSuchBucket, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}

	fmt.Println(result)
}

func putObject(content []byte) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-1"),
	})
	check(err)

	// S3 service
	svc := s3.New(sess)

	// input args to API call
	input := &s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(strings.NewReader(string(content))),
		Bucket: aws.String("d2a-config"),
		Key:    aws.String("foo.json"),
	}
	result, err := svc.PutObject(input)
	check(err)

	// TODO: put in a helper, this is duplicated with 'listObjects/0'
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				fmt.Println(s3.ErrCodeNoSuchBucket, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}

	fmt.Println(result)
}

// arn:${Partition}:amplify:${Region}:${Account}:apps/${AppId}/branches/${BranchName}

////////// File IO //////////

// reads and returns the file contents of 's'
func readFile(s string) []byte {
	fmt.Println("sss:", s, len(s))
	if strings.Contains(s, "TBD") {
		fmt.Println("File Contains TBD, skipping...")
		return []byte("")
	} else {
		dat, err := os.ReadFile(s)
		check(err)
		fmt.Print(string(dat))
		return []byte(dat)
	}
}

// prints a banner to std out
func printBanner(bannerName string) {
	line := strings.Repeat("-", LINE_LENGTH)
	fmt.Println(line)
	fmt.Println(bannerName)
	fmt.Println(line)

}

////////// Unmarshal //////////

// converts yaml bytes into a 'Manifest'
func unmarshlManifest(b []byte) Manifest {
	var m Manifest
	err := yaml.Unmarshal(b, &m)
	check(err)
	return m
}

////////// Validators //////////

func check(err error) {
	if err != nil {
		panic(err)
	}
}
