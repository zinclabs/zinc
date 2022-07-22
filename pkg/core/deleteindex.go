/* Copyright 2022 Zinc Labs Inc. and Contributors
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package core

import (
	"context"
	"errors"
	"os"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"

	"github.com/zinclabs/zinc/pkg/config"
	"github.com/zinclabs/zinc/pkg/metadata"
)

func DeleteIndex(name string) error {
	// 1. Check if index exists
	index, exists := GetIndex(name)
	if !exists {
		return errors.New("index " + name + " does not exists")
	}

	// 2. Close and Delete from cache
	ZINC_INDEX_LIST.Delete(name)

	// 3. Physically delete the index
	if index.GetStorageType() == "disk" {
		dataPath := config.Global.DataPath
		err := os.RemoveAll(dataPath + "/" + index.GetName())
		if err != nil {
			log.Error().Err(err).Msg("failed to delete index")
		}
	} else if index.GetStorageType() == "s3" {
		err := deleteFilesForIndexFromS3(index.GetName())
		if err != nil {
			log.Error().Err(err).Msg("failed to delete index from S3")
		}
	} else if index.GetStorageType() == "minio" {
		err := deleteFilesForIndexFromMinIO(index.GetName())
		if err != nil {
			log.Error().Err(err).Msg("failed to delete index from minIO")
		}
	}

	// 4. Delete form metadata
	return metadata.Index.Delete(name)
}

func deleteFilesForIndexFromMinIO(indexName string) error {
	endpoint := config.Global.MinIO.Endpoint
	accessKeyID := config.Global.MinIO.AccessKeyID
	secretAccessKey := config.Global.MinIO.SecretAccessKey
	minioBucket := config.Global.MinIO.Bucket

	opts := minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: false,
	}

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &opts)
	if err != nil {
		return err
	}

	listOpts := minio.ListObjectsOptions{
		Recursive: true,
		Prefix:    indexName,
	}

	objects := minioClient.ListObjects(context.TODO(), minioBucket, listOpts)

	for object := range objects {
		log.Print("Deleting: ", object.Key)
	}

	return nil
}

func deleteFilesForIndexFromS3(indexName string) error {
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Print("Error loading AWS config: ", err)
		return err
	}
	client := s3.NewFromConfig(cfg)

	s3bucket := config.Global.S3.Bucket
	ctx := context.Background()

	// List Objects in the bucket at prefix
	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket: &s3bucket,
		Prefix: &indexName,
	}
	listObjectsOutput, err := client.ListObjectsV2(ctx, listObjectsInput)
	if err != nil {
		log.Print("failed to list objects: ", err.Error())
		return err
	}

	var fileList []types.ObjectIdentifier

	for _, object := range listObjectsOutput.Contents {
		fileList = append(fileList, types.ObjectIdentifier{
			Key: object.Key,
		})
		log.Print("Deleting: ", *object.Key)
	}

	doi := &s3.DeleteObjectsInput{
		Bucket: &s3bucket,
		Delete: &types.Delete{
			Objects: fileList,
		},
	}
	_, err = client.DeleteObjects(ctx, doi)
	if err != nil {
		log.Print("failed to delete index: ", err.Error())
		return err
	}

	return nil
}
