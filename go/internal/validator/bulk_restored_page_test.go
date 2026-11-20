package validator_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestBulkRestoredPageCreateValidator_Validate(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	_, tx := testutil.SetupTx(t)
	v := validator.NewBulkRestoredPageCreateValidator(repository.NewPageRepository(testutil.QueriesWithTx(tx)))

	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("bulk-restored-page-validator").Build()
	otherSpaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("bulk-restored-page-validator-other").Build()
	topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(1).WithName("復元できる").Build()
	lockedTopicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(2).WithName("復元できない").Build()
	otherTopicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(otherSpaceID).WithNumber(1).WithName("別のスペース").Build()

	trashedAt := time.Now().Add(-time.Hour)
	newPage := func(space model.SpaceID, topic model.TopicID, number model.PageNumber) *testutil.PageBuilder {
		return testutil.NewPageBuilder(t, tx).WithSpaceID(space).WithTopicID(topic).WithNumber(number).WithTitle(fmt.Sprintf("ページ%d", number))
	}
	trashed := newPage(spaceID, topicID, 1).WithTrashedAt(trashedAt).Build()
	alreadyRestored := newPage(spaceID, topicID, 2).Build()
	locked := newPage(spaceID, lockedTopicID, 3).WithTrashedAt(trashedAt).Build()
	discarded := newPage(spaceID, topicID, 4).WithTrashedAt(trashedAt).WithDiscarded().Build()
	otherSpace := newPage(otherSpaceID, otherTopicID, 1).WithTrashedAt(trashedAt).Build()

	restorable := map[model.TopicID]bool{topicID: true}

	tests := []struct {
		name        string
		pageIDs     []string
		wantPageIDs []model.PageID
		wantError   bool
	}{
		{name: "ゴミ箱のページを返す", pageIDs: []string{string(trashed)}, wantPageIDs: []model.PageID{trashed}},
		{name: "同じIDを重ねて送っても1件として扱う", pageIDs: []string{string(trashed), string(trashed)}, wantPageIDs: []model.PageID{trashed}},
		{name: "大文字と小文字で重ねて送っても1件として扱う", pageIDs: []string{string(trashed), strings.ToUpper(string(trashed))}, wantPageIDs: []model.PageID{trashed}},
		{name: "urn:uuid:を付けたIDも同じページとして扱う", pageIDs: []string{"urn:uuid:" + string(trashed)}, wantPageIDs: []model.PageID{trashed}},
		{name: "既にゴミ箱から戻したページは、エラーにせず除く", pageIDs: []string{string(trashed), string(alreadyRestored)}, wantPageIDs: []model.PageID{trashed}},
		{name: "ページを選んでいなければエラー", pageIDs: nil, wantError: true},
		{name: "形式の誤ったIDはエラー", pageIDs: []string{"not-a-uuid"}, wantError: true},
		{name: "復元できないトピックのページを含むとエラー", pageIDs: []string{string(trashed), string(locked)}, wantError: true},
		{name: "廃棄したページを含むとエラー", pageIDs: []string{string(trashed), string(discarded)}, wantError: true},
		{name: "別のスペースのページを含むとエラー", pageIDs: []string{string(trashed), string(otherSpace)}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v.Validate(ctx, validator.BulkRestoredPageCreateValidatorInput{
				PageIDs:            tt.pageIDs,
				SpaceID:            spaceID,
				RestorableTopicIDs: restorable,
			})

			if tt.wantError {
				if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("page_ids") {
					t.Fatalf("page_idsのValidationErrorを期待したが、%vだった", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate()のエラー = %v", err)
			}
			if len(got) != len(tt.wantPageIDs) {
				t.Fatalf("件数 = %d、期待値 = %d", len(got), len(tt.wantPageIDs))
			}
			for i, id := range tt.wantPageIDs {
				if got[i].ID != id {
					t.Errorf("%d件目 = %v、期待値 = %v", i, got[i].ID, id)
				}
			}
		})
	}
}
