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

package search

import (
	"context"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/search"
	"golang.org/x/sync/errgroup"
)

func MultiSearch(ctx context.Context, req bluge.SearchRequest, readers ...*bluge.Reader) (search.DocumentMatchIterator, error) {
	if len(readers) == 0 {
		return nil, nil
	}
	if len(readers) == 1 {
		return readers[0].Search(ctx, req)
	}

	eg := &errgroup.Group{}
	eg.SetLimit(10)
	docs := make(chan *search.DocumentMatch, req.Collector().Size()*len(readers))
	aggs := make(chan *search.Bucket, len(readers))

	docList := &DocumentList{}
	eg2 := &errgroup.Group{}
	eg2.Go(func() error {
		for doc := range docs {
			docList.addDocument(doc)
		}
		return nil
	})
	eg2.Go(func() error {
		for agg := range aggs {
			if docList.bucket == nil {
				docList.bucket = agg
			} else {
				docList.bucket.Merge(agg)
			}
		}
		return nil
	})

	for _, r := range readers {
		r := r
		eg.Go(func() error {
			dmi, err := r.Search(ctx, req)
			if err != nil {
				return err
			}
			next, err := dmi.Next()
			for err == nil && next != nil {
				docs <- next
				next, err = dmi.Next()
			}
			aggs <- dmi.Aggregations()
			return err
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	close(docs)
	close(aggs)
	_ = eg2.Wait()

	docList.Done()

	return docList, nil
}

type DocumentList struct {
	docs   []*search.DocumentMatch
	bucket *search.Bucket
	next   int
}

func (d *DocumentList) addDocument(doc *search.DocumentMatch) {
	d.docs = append(d.docs, doc)
}

func (d *DocumentList) Done() {
	// sort
	// merge aggs
}

func (d *DocumentList) Next() (*search.DocumentMatch, error) {
	if d.next >= len(d.docs) {
		return nil, nil
	}
	doc := d.docs[d.next]
	d.next++
	return doc, nil
}

func (d *DocumentList) Aggregations() *search.Bucket {
	return d.bucket
}
