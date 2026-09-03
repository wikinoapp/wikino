package repository

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// TopicVisibility narrows a page listing down to the topics the viewer may open.
// The caller resolves the topic set with the same authorization rule the page screens use, so
// that the decision stays in the Policy layer instead of being expressed as scope names in SQL.
//
// [Ja] TopicVisibility はページ一覧を閲覧者が開けるトピックに絞り込む条件。
// トピックの集合はページ画面と同じ認可ルールで呼び出し元が解決する。
// これにより、判定が SQL 上のスコープ名ではなく Policy 層に残る。
type TopicVisibility struct {
	// AllVisible skips the narrowing entirely.
	//
	// [Ja] AllVisible は絞り込み自体を行わないことを表す。
	AllVisible bool

	// TopicIDs lists the visible topics. It is ignored when AllVisible is true.
	//
	// [Ja] TopicIDs は閲覧可能なトピックの一覧。AllVisible が true のときは無視される。
	TopicIDs []model.TopicID
}

// AllTopicsVisible returns a TopicVisibility that deliberately skips topic narrowing.
// It does not grant or prove authorization. It is currently used to preserve the edit screen's
// legacy behavior of showing page titles from every topic.
//
// [Ja] AllTopicsVisible は意図的にトピックの絞り込みを省略する TopicVisibility を返す。
// この値自体は認可を付与・保証しない。現在は編集画面で全トピックのページタイトルを見せる
// 従来挙動を維持するために使用する。
func AllTopicsVisible() TopicVisibility {
	return TopicVisibility{AllVisible: true}
}

// VisibleTopics returns a TopicVisibility limited to the given topics.
//
// [Ja] VisibleTopics は指定したトピックに限定した TopicVisibility を返す。
func VisibleTopics(topicIDs []model.TopicID) TopicVisibility {
	return TopicVisibility{TopicIDs: topicIDs}
}
