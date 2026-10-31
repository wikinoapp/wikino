package usecase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"testing/iotest"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// testPNGは、中身から判別するとPNGになるバイト列 (PNGのシグネチャで始まる)
var testPNG = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...)

// testJPEGは、中身から判別するとJPEGになるバイト列 (JPEGのSOIマーカーで始まる)
var testJPEG = append([]byte("\xff\xd8\xff"), bytes.Repeat([]byte{0}, 64)...)

func TestGetAPIAttachmentContentUsecase_Execute(t *testing.T) {
	t.Parallel()

	// 添付ファイルのフィクスチャは、記録された種類がimage/pngで、記録されたバイト数が2048である
	tests := []struct {
		name            string
		tokenScopes     []model.Scope
		content         []byte
		contentTypes    []string
		maxBytes        int64
		wantContent     bool
		wantContentType string
		wantNotFound    bool
	}{
		{
			name:            "種類と大きさの条件を満たす画像は中身を返す",
			tokenScopes:     []model.Scope{model.ScopeAttachmentRead},
			content:         testPNG,
			contentTypes:    []string{"image/png"},
			maxBytes:        4096,
			wantContent:     true,
			wantContentType: "image/png",
		},
		{
			// 記録はPNGだが中身はJPEGで、両方が対象の種類に含まれる
			name:            "記録と中身の種類が違えば中身から判別した種類で返す",
			tokenScopes:     []model.Scope{model.ScopeAttachmentRead},
			content:         testJPEG,
			contentTypes:    []string{"image/png", "image/jpeg"},
			maxBytes:        4096,
			wantContent:     true,
			wantContentType: "image/jpeg",
		},
		{
			name:         "記録された種類が対象でなければ中身を返さない",
			tokenScopes:  []model.Scope{model.ScopeAttachmentRead},
			content:      testPNG,
			contentTypes: []string{"image/jpeg"},
			maxBytes:     4096,
		},
		{
			// 記録されたバイト数 (2048) が上限を超えるため、ストレージを読まない (中身を置かなくても失敗しない)
			name:         "記録されたバイト数が上限を超えれば中身を返さない",
			tokenScopes:  []model.Scope{model.ScopeAttachmentRead},
			contentTypes: []string{"image/png"},
			maxBytes:     1024,
		},
		{
			name:         "実際の中身が上限を超えれば中身を返さない",
			tokenScopes:  []model.Scope{model.ScopeAttachmentRead},
			content:      append(append([]byte{}, testPNG...), bytes.Repeat([]byte{0}, 5000)...),
			contentTypes: []string{"image/png"},
			maxBytes:     4096,
		},
		{
			name:         "中身から判別した種類が対象でなければ中身を返さない",
			tokenScopes:  []model.Scope{model.ScopeAttachmentRead},
			content:      []byte("<html><script>alert(1)</script></html>"),
			contentTypes: []string{"image/png"},
			maxBytes:     4096,
		},
		{
			name:         "トークンがattachment:readを持たなければ見つからない",
			tokenScopes:  []model.Scope{model.ScopePageRead},
			content:      testPNG,
			contentTypes: []string{"image/png"},
			maxBytes:     4096,
			wantNotFound: true,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			identifier := "attachment-content-" + string(rune('a'+i))
			f := setupAPIAttachmentFixture(t, tx, identifier, apiTopicRegularMemberRole)
			objectStorage := storage.NewFakeObjectStorage()
			if tt.content != nil {
				objectStorage.Put(identifier+"-public", tt.content, "image/png")
			}

			uc := NewGetAPIAttachmentContentUsecase(repository.NewAttachmentRepository(testutil.QueriesWithTx(tx)), objectStorage)
			output, err := uc.Execute(t.Context(), GetAPIAttachmentContentInput{
				Principal:       f.principal(apiTopicRegularMemberRole, tt.tokenScopes),
				SpaceIdentifier: model.SpaceIdentifier(identifier),
				AttachmentID:    f.attachmentIDs["public"],
				ContentTypes:    tt.contentTypes,
				MaxBytes:        tt.maxBytes,
			})

			if tt.wantNotFound {
				if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
					t.Errorf("エラー = %v、期待値 = AppErrCodeResourceNotFound", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if output.Attachment == nil || output.Attachment.ID != f.attachmentIDs["public"] {
				t.Errorf("Attachment = %+v、期待値 = 指定した添付ファイル", output.Attachment)
			}
			if tt.wantContent {
				if !bytes.Equal(output.Content, tt.content) || output.ContentType != tt.wantContentType {
					t.Errorf("中身 = %dバイト (%s)、期待値 = 置いた中身 (%s)", len(output.Content), output.ContentType, tt.wantContentType)
				}
				return
			}
			if output.Content != nil || output.ContentType != "" {
				t.Errorf("中身 = %dバイト (%q)、期待値 = 返さない", len(output.Content), output.ContentType)
			}
		})
	}
}

// attachmentContentReadCloserは、実際に読んだ量とストリームを閉じたことを記録する。
// 結果を返さないだけでなく、ストレージの読み取り量そのものが制限されることを確かめるために使う
type attachmentContentReadCloser struct {
	reader    io.Reader
	bytesRead int64
	closed    bool
}

func (r *attachmentContentReadCloser) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.bytesRead += int64(n)
	return n, err
}

func (r *attachmentContentReadCloser) Close() error {
	r.closed = true
	return nil
}

// attachmentContentObjectStorageは、他の操作は既存のFakeObjectStorageに任せ、Getで観測用のストリームを返す
type attachmentContentObjectStorage struct {
	*storage.FakeObjectStorage
	body io.ReadCloser
}

func (s *attachmentContentObjectStorage) Get(context.Context, string) (io.ReadCloser, error) {
	return s.body, nil
}

func TestGetAPIAttachmentContentUsecase_読み取り量を制限してストリームを閉じる(t *testing.T) {
	t.Parallel()

	const maxBytes int64 = 4096
	readErr := errors.New("ストリームを読めない")
	tests := []struct {
		name          string
		reader        io.Reader
		wantBytesRead int64
		wantContent   bool
		wantErr       error
	}{
		{
			name: "正常な画像", reader: bytes.NewReader(testPNG),
			wantBytesRead: int64(len(testPNG)), wantContent: true,
		},
		{
			name: "上限と同じ大きさの画像", reader: io.MultiReader(bytes.NewReader(testPNG), bytes.NewReader(make([]byte, int(maxBytes)-len(testPNG)))),
			wantBytesRead: maxBytes, wantContent: true,
		},
		{
			// DBの記録は2048バイトだが、実体は上限を超えている
			name: "上限超過の判定に必要な1バイトまで読む", reader: io.MultiReader(bytes.NewReader(testPNG), bytes.NewReader(make([]byte, maxBytes*2))),
			wantBytesRead: maxBytes + 1,
		},
		{
			name: "読み取りエラー", reader: iotest.ErrReader(readErr), wantErr: readErr,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			identifier := "attachment-stream-" + string(rune('a'+i))
			f := setupAPIAttachmentFixture(t, tx, identifier, apiTopicRegularMemberRole)
			body := &attachmentContentReadCloser{reader: tt.reader}
			objectStorage := &attachmentContentObjectStorage{FakeObjectStorage: storage.NewFakeObjectStorage(), body: body}
			uc := NewGetAPIAttachmentContentUsecase(repository.NewAttachmentRepository(testutil.QueriesWithTx(tx)), objectStorage)

			output, err := uc.Execute(t.Context(), GetAPIAttachmentContentInput{
				Principal:       f.principal(apiTopicRegularMemberRole, []model.Scope{model.ScopeAttachmentRead}),
				SpaceIdentifier: model.SpaceIdentifier(identifier),
				AttachmentID:    f.attachmentIDs["public"],
				ContentTypes:    []string{"image/png"},
				MaxBytes:        maxBytes,
			})

			if body.bytesRead != tt.wantBytesRead {
				t.Errorf("読んだ量 = %dバイト、期待値 = %dバイト", body.bytesRead, tt.wantBytesRead)
			}
			if !body.closed {
				t.Error("ストリームが閉じられていない")
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || output != nil {
					t.Errorf("output = %+v、エラー = %v、期待値 = nilと読み取りエラー", output, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if (output.Content != nil) != tt.wantContent {
				t.Errorf("中身 = %dバイト、中身を返すことの期待値 = %v", len(output.Content), tt.wantContent)
			}
		})
	}
}
