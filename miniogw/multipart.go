// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package miniogw

import (
	"context"
	"crypto/md5" /* #nosec G501 */ // Is only used for calculating a hash of the ETags of the all the parts of a multipart upload.
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

	minio "github.com/minio/minio/cmd"

	"storj.io/uplink"
)

func (layer *gatewayLayer) NewMultipartUpload(ctx context.Context, bucket, object string, opts minio.ObjectOptions) (uploadID string, err error) {
	defer mon.Task()(&ctx)(&err)

	info, err := layer.project.NewMultipartUpload(ctx, bucket, object, nil)
	if err != nil {
		return "", convertMultipartError(err, bucket, object, "")
	}
	return info.StreamID, nil
}

func (layer *gatewayLayer) GetMultipartInfo(ctx context.Context, bucket string, object string, uploadID string, opts minio.ObjectOptions) (info minio.MultipartInfo, err error) {
	info.Bucket = bucket
	info.Object = object
	info.UploadID = uploadID
	// TODO: We need an uplink API for this
	return info, nil
}

func (layer *gatewayLayer) PutObjectPart(ctx context.Context, bucket, object, uploadID string, partID int, data *minio.PutObjReader, opts minio.ObjectOptions) (info minio.PartInfo, err error) {
	defer mon.Task()(&ctx)(&err)

	partInfo, err := layer.project.PutObjectPart(ctx, bucket, object, uploadID, partID, data)
	if err != nil {
		return minio.PartInfo{}, convertMultipartError(err, bucket, object, uploadID)
	}

	// TODO: Store the part's ETag in metabase

	return minio.PartInfo{
		PartNumber: partID,
		Size:       partInfo.Size,
		ETag:       data.MD5CurrentHexString(),
	}, nil
}

func (layer *gatewayLayer) AbortMultipartUpload(ctx context.Context, bucket, object, uploadID string, _ minio.ObjectOptions) (err error) {
	defer mon.Task()(&ctx)(&err)
	err = layer.project.AbortMultipartUpload(ctx, bucket, object, uploadID)
	if err != nil {
		return convertMultipartError(err, bucket, object, uploadID)
	}
	return nil
}

func (layer *gatewayLayer) CompleteMultipartUpload(ctx context.Context, bucket, object, uploadID string, uploadedParts []minio.CompletePart, opts minio.ObjectOptions) (objInfo minio.ObjectInfo, err error) {
	defer mon.Task()(&ctx)(&err)

	// TODO: Check that ETag of uploadedParts match the ETags stored in metabase.

	etag, err := multipartUploadETag(uploadedParts)
	if err != nil {
		return minio.ObjectInfo{}, convertMultipartError(err, bucket, object, uploadID)
	}

	metadata := uplink.CustomMetadata(opts.UserDefined).Clone()
	metadata["s3:etag"] = etag

	obj, err := layer.project.CompleteMultipartUpload(ctx, bucket, object, uploadID, &uplink.MultipartObjectOptions{
		CustomMetadata: metadata,
	})
	if err != nil {
		return minio.ObjectInfo{}, convertMultipartError(err, bucket, object, uploadID)
	}

	return minioObjectInfo(bucket, etag, obj), nil
}

func (layer *gatewayLayer) ListObjectParts(ctx context.Context, bucket, object, uploadID string, partNumberMarker int, maxParts int, opts minio.ObjectOptions) (result minio.ListPartsInfo, err error) {
	defer mon.Task()(&ctx)(&err)

	list, err := layer.project.ListObjectParts(ctx, bucket, object, uploadID, partNumberMarker, maxParts)
	if err != nil {
		return minio.ListPartsInfo{}, convertMultipartError(err, bucket, object, uploadID)
	}

	parts := make([]minio.PartInfo, 0, len(list.Items))
	for _, item := range list.Items {
		parts = append(parts, minio.PartInfo{
			PartNumber:   item.PartNumber,
			LastModified: item.LastModified,
			ETag:         "",        // TODO: Entity tag returned when the part was initially uploaded.
			Size:         item.Size, // Size in bytes of the part.
			ActualSize:   item.Size, // Decompressed Size.
		})
	}
	return minio.ListPartsInfo{
		Bucket:               bucket,
		Object:               object,
		UploadID:             uploadID,
		StorageClass:         "",               // TODO
		PartNumberMarker:     partNumberMarker, // Part number after which listing begins.
		NextPartNumberMarker: partNumberMarker, // TODO Next part number marker to be used if list is truncated
		MaxParts:             maxParts,
		IsTruncated:          list.More,
		Parts:                parts,
		// also available: UserDefined map[string]string
	}, nil
}

// ListMultipartUploads lists all multipart uploads.
func (layer *gatewayLayer) ListMultipartUploads(ctx context.Context, bucket string, prefix string, keyMarker string, uploadIDMarker string, delimiter string, maxUploads int) (result minio.ListMultipartsInfo, err error) {
	defer mon.Task()(&ctx)(&err)

	// TODO maybe this should be checked by project.ListMultipartUploads
	if bucket == "" {
		return minio.ListMultipartsInfo{}, minio.BucketNameInvalid{}
	}

	if delimiter != "" && delimiter != "/" {
		return minio.ListMultipartsInfo{}, minio.UnsupportedDelimiter{Delimiter: delimiter}
	}

	// TODO this should be removed and implemented on satellite side
	_, err = layer.project.StatBucket(ctx, bucket)
	if err != nil {
		return minio.ListMultipartsInfo{}, convertMultipartError(err, bucket, "", "")
	}

	recursive := delimiter == ""

	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		// N.B.: in this case, the most S3-compatible thing we could do
		// is ask the satellite to list all siblings of this prefix that
		// share the same parent encryption key, decrypt all of them,
		// then only return the ones that have this same unencrypted
		// prefix.
		// this is terrible from a performance perspective, and it turns
		// out, many of the usages of listing without a /-suffix are
		// simply to provide a sort of StatObject like feature. in fact,
		// for example, duplicity never calls list without a /-suffix
		// in a case where it expects to get back more than one result.
		// so, we could either
		// 1) return an error here, guaranteeing nothing works
		// 2) do the full S3 compatible thing, which has terrible
		//    performance for a really common case (StatObject-like
		//		functionality)
		// 3) handle strictly more of the use cases than #1 without
		//    loss of performance by turning this into a StatObject.
		// so we do #3 here. it's great!

		return layer.listSingleUpload(ctx, bucket, prefix, recursive)
	}

	list := layer.project.ListMultipartUploads(ctx, bucket, &uplink.ListMultipartUploadsOptions{
		Prefix:    prefix,
		Cursor:    keyMarker,
		Recursive: recursive,

		System: true,
		Custom: true,
	})

	startAfter := keyMarker
	var uploads []minio.MultipartInfo
	var prefixes []string

	limit := maxUploads
	for (limit > 0 || maxUploads == 0) && list.Next() {
		limit--
		object := list.Item()
		if object.IsPrefix {
			prefixes = append(prefixes, object.Key)
			continue
		}

		uploads = append(uploads, minioMultipartInfo(bucket, object))

		startAfter = object.Key

	}
	if list.Err() != nil {
		return result, convertMultipartError(list.Err(), bucket, "", "")
	}

	more := list.Next()
	if list.Err() != nil {
		return result, convertMultipartError(list.Err(), bucket, "", "")
	}

	result = minio.ListMultipartsInfo{
		KeyMarker:      keyMarker,
		UploadIDMarker: uploadIDMarker,
		MaxUploads:     maxUploads,
		IsTruncated:    more,
		Uploads:        uploads,
		Prefix:         prefix,
		Delimiter:      delimiter,
		CommonPrefixes: prefixes,
	}
	if more {
		result.NextKeyMarker = startAfter
		// TODO: NextUploadID
	}

	return result, nil
}

func (layer *gatewayLayer) listSingleUpload(ctx context.Context, bucketName, key string, recursive bool) (result minio.ListMultipartsInfo, err error) {
	defer mon.Task()(&ctx)(&err)

	var prefixes []string
	if !recursive {
		list := layer.project.ListMultipartUploads(ctx, bucketName, &uplink.ListMultipartUploadsOptions{
			Prefix:    key + "/",
			Recursive: true,
			// Limit: 1, would be nice to set here
		})
		if list.Next() {
			prefixes = append(prefixes, key+"/")
		}
		if err := list.Err(); err != nil {
			return minio.ListMultipartsInfo{}, convertMultipartError(err, bucketName, key, "")
		}
	}

	var uploads []minio.MultipartInfo
	// TODO: we need a uplink API to list the pending uploads for a specific object key
	// upload, err := layer.project.StatObject(ctx, bucketName, key)
	// if err != nil {
	// 	if !errors.Is(err, uplink.ErrObjectNotFound) {
	// 		return minio.ListMultipartsInfo{}, convertMultipartError(err, bucketName, key, "")
	// 	}
	// } else {
	// 	uploads = append(uploads, minioObjectInfo(bucketName, "", upload))
	// }

	return minio.ListMultipartsInfo{
		IsTruncated:    false,
		CommonPrefixes: prefixes,
		Uploads:        uploads,
	}, nil
}

func minioMultipartInfo(bucket string, object *uplink.Object) minio.MultipartInfo {
	if object == nil {
		object = &uplink.Object{}
	}

	return minio.MultipartInfo{
		Bucket:      bucket,
		Object:      object.Key,
		Initiated:   object.System.Created,
		UploadID:    object.System.StreamID,
		UserDefined: object.Custom,
	}
}

func multipartUploadETag(parts []minio.CompletePart) (string, error) {
	var hashes []byte
	for _, part := range parts {
		md5, err := hex.DecodeString(canonicalEtag(part.ETag))
		if err != nil {
			hashes = append(hashes, []byte(part.ETag)...)
		} else {
			hashes = append(hashes, md5...)
		}
	}

	/* #nosec G401 */ // ETags aren't security sensitive
	sum := md5.Sum(hashes)
	return hex.EncodeToString(sum[:]) + "-" + strconv.Itoa(len(parts)), nil
}

func canonicalEtag(etag string) string {
	etag = strings.Trim(etag, `"`)
	p := strings.IndexByte(etag, '-')
	if p >= 0 {
		return etag[:p]
	}
	return etag
}

func convertMultipartError(err error, bucket, object, uploadID string) error {
	if errors.Is(err, uplink.ErrStreamIDInvalid) {
		return minio.InvalidUploadID{Bucket: bucket, Object: object, UploadID: uploadID}
	}

	return convertError(err, bucket, object)
}
