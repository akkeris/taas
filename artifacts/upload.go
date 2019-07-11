package artifacts
import (
        "github.com/aws/aws-sdk-go/aws"
        "github.com/aws/aws-sdk-go/aws/awserr"
        "github.com/aws/aws-sdk-go/aws/session"
        "github.com/aws/aws-sdk-go/service/s3"
        "fmt"
        "bytes"
        "os"
)


func UploadToS3(body string, contenttype string, runid string) {
stringBytes := bytes.NewReader([]byte(body))

        svc := s3.New(session.New(&aws.Config{
                Region: aws.String(os.Getenv("AWS_REGION")),
        }))

input := &s3.PutObjectInput{
    Body:   aws.ReadSeekCloser(stringBytes),
    Bucket: aws.String(os.Getenv("AWS_S3_BUCKET")),
    Key:    aws.String(runid+"/describe.txt"),
    ContentType: aws.String(contenttype),
}

result, err := svc.PutObject(input)
if err != nil {
    if aerr, ok := err.(awserr.Error); ok {
        switch aerr.Code() {
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
