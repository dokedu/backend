package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.29

import (
	"context"
	"database/sql"
	"errors"
	"example/pkg/db"
	"example/pkg/graph/model"
	"example/pkg/middleware"
	"fmt"
	"strings"
	"time"

	minio "github.com/minio/minio-go/v7"
)

// User is the resolver for the user field.
func (r *bucketResolver) User(ctx context.Context, obj *db.Bucket) (*db.User, error) {
	currentUser := middleware.ForContext(ctx)
	if currentUser == nil {
		return nil, errors.New("no user found in the context")
	}

	var user db.User
	err := r.DB.NewSelect().Model(&user).Where("id = ?", obj.UserID).Where("organisation_id = ?", currentUser.OrganisationID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// DeletedAt is the resolver for the deletedAt field.
func (r *bucketResolver) DeletedAt(ctx context.Context, obj *db.Bucket) (*time.Time, error) {
	panic(fmt.Errorf("not implemented: DeletedAt - deletedAt"))
}

// Files is the resolver for the files field.
func (r *bucketResolver) Files(ctx context.Context, obj *db.Bucket) ([]*db.File, error) {
	currentUser := middleware.ForContext(ctx)
	if currentUser == nil {
		return nil, errors.New("no user found in the context")
	}

	var files []*db.File
	err := r.DB.NewSelect().Model(&files).Where("bucket_id = ?", obj.ID).Where("organisation_id = ?", currentUser.OrganisationID).Order("name").Scan(ctx)
	if err != nil {
		return nil, err
	}

	return files, nil
}

// Bucket is the resolver for the bucket field.
func (r *fileResolver) Bucket(ctx context.Context, obj *db.File) (*db.Bucket, error) {
	currentUser := middleware.ForContext(ctx)
	if currentUser == nil {
		return nil, errors.New("no user found in the context")
	}

	var bucket db.Bucket
	err := r.DB.NewSelect().Model(&bucket).Where("id = ?", obj.BucketID).Where("organisation_id = ?", currentUser.OrganisationID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &bucket, nil
}

// Parent is the resolver for the parent field.
func (r *fileResolver) Parent(ctx context.Context, obj *db.File) (*db.File, error) {
	currentUser := middleware.ForContext(ctx)
	if currentUser == nil {
		return nil, errors.New("no user found in the context")
	}

	if !obj.ParentID.Valid {
		return nil, nil
	}

	var parent db.File
	err := r.DB.NewSelect().Model(&parent).Where("id = ?", obj.ParentID.String).Where("organisation_id = ?", currentUser.OrganisationID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &parent, nil
}

// DeletedAt is the resolver for the deletedAt field.
func (r *fileResolver) DeletedAt(ctx context.Context, obj *db.File) (*time.Time, error) {
	if !obj.DeletedAt.IsZero() {
		return nil, nil
	}

	deletedAt := obj.DeletedAt.Time
	return &deletedAt, nil
}

// Parents is the resolver for the parents field.
func (r *fileResolver) Parents(ctx context.Context, obj *db.File) ([]*db.File, error) {
	currentUser := middleware.ForContext(ctx)
	if currentUser == nil {
		return nil, errors.New("no user found in the context")
	}

	query := `
WITH RECURSIVE file_parents AS (
    SELECT *, 1 AS level
    FROM files
    WHERE id = ? AND organisation_id = ?
    
    UNION ALL
    
    SELECT f.*, fp.level + 1
    FROM files f
    JOIN file_parents fp ON f.id = fp.parent_id
	WHERE f.organisation_id = ?
)
SELECT file_parents.id, file_parents.name, file_parents.file_type, file_parents.mime_type, file_parents.size, file_parents.bucket_id, file_parents.parent_id, file_parents.organisation_id, file_parents.created_at, file_parents.deleted_at
FROM file_parents
WHERE id <> ?
ORDER BY level DESC;
`

	// query without new lines
	q := strings.ReplaceAll(query, "\n", " ")

	var files []*db.File
	err := r.DB.NewRaw(q, obj.ID, currentUser.OrganisationID, currentUser.OrganisationID, obj.ID).Scan(ctx, &files)
	if err != nil {
		return nil, err
	}

	return files, nil
}

// Files is the resolver for the files field.
func (r *fileResolver) Files(ctx context.Context, obj *db.File) ([]*db.File, error) {
	currentUser := middleware.ForContext(ctx)
	if currentUser == nil {
		return nil, errors.New("no user found in the context")
	}

	var files []*db.File
	err := r.DB.NewSelect().Model(&files).Where("parent_id = ?", obj.ID).Where("organisation_id = ?", currentUser.OrganisationID).Order("name").Scan(ctx)
	if err != nil {
		return nil, err
	}

	return files, nil
}

// SingleUpload is the resolver for the singleUpload field.
func (r *mutationResolver) SingleUpload(ctx context.Context, input model.FileUploadInput) (*db.File, error) {
	currentUser := middleware.ForContext(ctx)
	if currentUser == nil {
		return nil, errors.New("no user found in the context")
	}

	var file db.File
	file.Name = input.File.Filename
	file.FileType = "blob"
	file.OrganisationID = currentUser.OrganisationID
	file.Size = input.File.Size

	var bucket db.Bucket

	if input.BucketID != nil && len(*input.BucketID) > 0 {
		file.BucketID = *input.BucketID
	} else {
		err := r.DB.NewSelect().Model(&bucket).Column("id").Where("user_id = ?", currentUser.ID).Scan(ctx)

		if err != nil && err.Error() == "sql: no rows in result set" {
			// create bucket for user
			bucket.Name = "User Bucket " + currentUser.ID
			bucket.UserID = sql.NullString{String: currentUser.ID, Valid: true}
			bucket.OrganisationID = currentUser.OrganisationID
			err = r.DB.NewInsert().Model(&bucket).Returning("*").Scan(ctx)
			if err != nil {
				return nil, err
			}

			err := r.MinioClient.MakeBucket(ctx, bucket.ID, minio.MakeBucketOptions{})
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}

		file.BucketID = bucket.ID
	}

	if input.ParentID != nil && len(*input.ParentID) > 0 {
		file.ParentID = sql.NullString{String: *input.ParentID, Valid: true}
	}

	err := r.DB.NewInsert().Model(&file).Returning("*").Scan(ctx)
	if err != nil {
		return nil, err
	}

	// Upload the file to specific bucket with the file id
	_, err = r.MinioClient.PutObject(ctx, bucket.ID, file.ID, input.File.File, input.File.Size, minio.PutObjectOptions{
		ContentType: input.File.ContentType,
	})
	if err != nil {
		return nil, err
	}

	return &file, nil
}

// CreateFolder is the resolver for the createFolder field.
func (r *mutationResolver) CreateFolder(ctx context.Context, input model.CreateFolderInput) (*db.File, error) {
	currentUser := middleware.ForContext(ctx)
	if currentUser == nil {
		return nil, errors.New("no user found in the context")
	}

	var file db.File
	file.Name = input.Name
	file.FileType = "folder"
	file.OrganisationID = currentUser.OrganisationID

	if input.BucketID != nil && len(*input.BucketID) > 0 {
		file.BucketID = *input.BucketID
	} else {
		var bucket db.Bucket
		err := r.DB.NewSelect().Model(&bucket).Column("id").Where("user_id = ?", currentUser.ID).Scan(ctx)

		if err != nil && err.Error() == "sql: no rows in result set" {
			// create bucket for user
			bucket.Name = "User Bucket " + currentUser.ID
			bucket.UserID = sql.NullString{String: currentUser.ID, Valid: true}
			bucket.OrganisationID = currentUser.OrganisationID
			err = r.DB.NewInsert().Model(&bucket).Returning("*").Scan(ctx)
			if err != nil {
				return nil, err
			}
			err := r.MinioClient.MakeBucket(ctx, bucket.ID, minio.MakeBucketOptions{})
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}

		file.BucketID = bucket.ID
	}

	if input.ParentID != nil && len(*input.ParentID) > 0 {
		file.ParentID = sql.NullString{String: *input.ParentID, Valid: true}
	}

	err := r.DB.NewInsert().Model(&file).Returning("*").Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &file, nil
}

// GenerateFileURL is the resolver for the generateFileURL field.
func (r *mutationResolver) GenerateFileURL(ctx context.Context, input model.GenerateFileURLInput) (*model.GenerateFileURLPayload, error) {
	currentUser := middleware.ForContext(ctx)
	if currentUser == nil {
		return nil, errors.New("no user found in the context")
	}

	var file db.File
	err := r.DB.NewSelect().Model(&file).Where("id = ?", input.ID).Where("organisation_id = ?", currentUser.OrganisationID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	presignedURL, err := r.MinioClient.PresignedGetObject(ctx, file.BucketID, file.ID, time.Second*60, nil)

	if err != nil {
		return nil, err
	}

	return &model.GenerateFileURLPayload{URL: presignedURL.String()}, nil
}

// Buckets is the resolver for the buckets field.
func (r *queryResolver) Buckets(ctx context.Context, input *model.BucketFilterInput) (*model.BucketConnection, error) {
	currentUser := middleware.ForContext(ctx)
	if currentUser == nil {
		return nil, errors.New("no user found in the context")
	}

	var buckets []*db.Bucket
	query := r.DB.NewSelect().Model(&buckets).Where("user_id = ?", currentUser.ID)

	if input != nil {
		if input.Shared != nil {
			query.Where("shared = ?", *input.Shared)
		}
	}

	count, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &model.BucketConnection{
		Edges:      buckets,
		TotalCount: count,
	}, nil
}

// Bucket is the resolver for the bucket field.
func (r *queryResolver) Bucket(ctx context.Context, id string) (*db.Bucket, error) {
	currentUser := middleware.ForContext(ctx)
	if currentUser == nil {
		return nil, errors.New("no user found in the context")
	}

	var bucket db.Bucket
	err := r.DB.NewSelect().Model(&bucket).Where("id = ?", id).Where("user_id = ?", currentUser.ID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &bucket, nil
}

// File is the resolver for the file field.
func (r *queryResolver) File(ctx context.Context, id string) (*db.File, error) {
	currentUser := middleware.ForContext(ctx)
	if currentUser == nil {
		return nil, errors.New("no user found in the context")
	}

	var file db.File
	err := r.DB.NewSelect().Model(&file).Where("id = ?", id).Where("organisation_id = ?", currentUser.OrganisationID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &file, nil
}

// Files is the resolver for the files field.
func (r *queryResolver) Files(ctx context.Context, input *model.FilesFilterInput) (*model.FileConnection, error) {
	currentUser := middleware.ForContext(ctx)
	if currentUser == nil {
		return nil, errors.New("no user found in the context")
	}

	var files []*db.File
	query := r.DB.NewSelect().Model(&files).Where("organisation_id = ?", currentUser.OrganisationID).Order("name")

	if input != nil {
		if input.ParentID != nil && len(*input.ParentID) > 0 {
			query.Where("parent_id = ?", *input.ParentID)
		} else {
			query.Where("parent_id IS NULL")
		}
		if input.BucketID != nil && len(*input.BucketID) > 0 {
			query.Where("bucket_id = ?", *input.BucketID)
		}
		if input.Limit != nil {
			query.Limit(*input.Limit)
		}
		if input.Offset != nil {
			query.Offset(*input.Offset)
		}
	}

	count, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &model.FileConnection{
		Edges:      files,
		TotalCount: count,
		PageInfo:   nil,
	}, nil
}

// MyFiles is the resolver for the myFiles field.
func (r *queryResolver) MyFiles(ctx context.Context, input *model.MyFilesFilterInput) (*model.FileConnection, error) {
	currentUser := middleware.ForContext(ctx)
	if currentUser == nil {
		return nil, errors.New("no user found in the context")
	}

	var bucket db.Bucket
	err := r.DB.NewSelect().Model(&bucket).Column("id").Where("user_id = ?", currentUser.ID).Scan(ctx)

	var files []*db.File
	query := r.DB.NewSelect().Model(&files).Where("organisation_id = ?", currentUser.OrganisationID).Where("bucket_id = ?", bucket.ID).Order("name")

	if input != nil {
		if input.ParentID != nil && len(*input.ParentID) > 0 {
			query.Where("parent_id = ?", *input.ParentID)
		} else {
			query.Where("parent_id IS NULL")
		}
		if input.Limit != nil {
			query.Limit(*input.Limit)
		}
		if input.Offset != nil {
			query.Offset(*input.Offset)
		}
	}

	count, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &model.FileConnection{
		Edges:      files,
		TotalCount: count,
		PageInfo:   nil,
	}, nil
}

// MyBucket is the resolver for the myBucket field.
func (r *queryResolver) MyBucket(ctx context.Context, id string) (*db.Bucket, error) {
	panic(fmt.Errorf("not implemented: MyBucket - myBucket"))
}

// Bucket returns BucketResolver implementation.
func (r *Resolver) Bucket() BucketResolver { return &bucketResolver{r} }

// File returns FileResolver implementation.
func (r *Resolver) File() FileResolver { return &fileResolver{r} }

type bucketResolver struct{ *Resolver }
type fileResolver struct{ *Resolver }
